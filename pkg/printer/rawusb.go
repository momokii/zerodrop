package printer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/zerodrop/terminal/pkg/qr"
)

// usbfs ioctl constants (Linux x86_64)
const (
	usbDevfsClaimInterface   = 0x8004550f
	usbDevfsReleaseInterface = 0x80045510
	usbDevfsBulk             = 0xc0185502
	usbDevfsSetConfiguration = 0x80045505
)

// usbBulkTransfer matches struct usbdevfs_bulktransfer from <linux/usbdevice_fs.h>
// On x86_64: ep(4) + len(4) + timeout(4) + pad(4) + data(8) = 24 bytes
type usbBulkTransfer struct {
	Ep      uint32
	Len     uint32
	Timeout uint32
	_       uint32
	Data    unsafe.Pointer
}

// RawUSBPrinter communicates directly with a USB thermal printer via usbfs,
// bypassing the usblp kernel driver. This is needed for USB-to-parallel bridge
// chips (e.g. Zjiang 0fe6:811e) that expose Bulk OUT + Interrupt IN instead of
// Bulk OUT + Bulk IN, which causes usblp's probe to reject them.
type RawUSBPrinter struct {
	busNum     int
	devNum     int
	devicePath string // /dev/bus/usb/BBB/DDD
	sysfsPath  string // /sys/bus/usb/devices/X-Y/
	ifaceNum   int
	bulkOutEp  uint8
	available  bool
	mu         sync.Mutex
}

// NewRawUSBPrinter finds a USB thermal printer by VID:PID and prepares
// raw USB access via usbfs. Does NOT open the device until Print is called.
func NewRawUSBPrinter(vid, pid string) (*RawUSBPrinter, error) {
	sysfsPath, busNum, devNum, bulkOutEp, ifaceNum, err := scanUSBDevice(vid, pid)
	if err != nil {
		return nil, err
	}

	devicePath := fmt.Sprintf("/dev/bus/usb/%03d/%03d", busNum, devNum)

	p := &RawUSBPrinter{
		busNum:     busNum,
		devNum:     devNum,
		devicePath: devicePath,
		sysfsPath:  sysfsPath,
		ifaceNum:   ifaceNum,
		bulkOutEp:  bulkOutEp,
		available:  true,
	}

	return p, nil
}

// Print opens the USB device, claims the interface, sends ESC/POS data via
// bulk transfer to the OUT endpoint, then releases the interface.
func (p *RawUSBPrinter) Print(ciphertext []byte) error {
	if !p.available {
		return fmt.Errorf("raw USB printer is not available")
	}

	// Generate QR ESC/POS commands
	escpos, err := qr.GenerateQRESCPOS(ciphertext)
	if err != nil {
		return fmt.Errorf("failed to generate QR: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	fd, err := syscall.Open(p.devicePath, syscall.O_RDWR, 0)
	if err != nil {
		p.available = false
		return fmt.Errorf("cannot open USB device %s: %w", p.devicePath, err)
	}
	defer syscall.Close(fd)

	// Set configuration 1
	config := uint32(1)
	if err := ioctl(uintptr(fd), usbDevfsSetConfiguration, uintptr(unsafe.Pointer(&config))); err != nil {
		// Non-fatal: device may already be configured
		log.Printf("[RawUSB] set configuration: %v (continuing)", err)
	}

	// Claim interface
	iface := uint32(p.ifaceNum)
	if err := ioctl(uintptr(fd), usbDevfsClaimInterface, uintptr(unsafe.Pointer(&iface))); err != nil {
		return fmt.Errorf("cannot claim USB interface %d: %w", p.ifaceNum, err)
	}
	defer func() {
		releaseIf := uint32(p.ifaceNum)
		if err := ioctl(uintptr(fd), usbDevfsReleaseInterface, uintptr(unsafe.Pointer(&releaseIf))); err != nil {
			log.Printf("[RawUSB] release interface: %v", err)
		}
	}()

	// Send ESC/POS data in chunks (max 4096 per bulk transfer)
	const chunkSize = 4096
	for offset := 0; offset < len(escpos); offset += chunkSize {
		end := offset + chunkSize
		if end > len(escpos) {
			end = len(escpos)
		}
		chunk := escpos[offset:end]

		xfer := usbBulkTransfer{
			Ep:      uint32(p.bulkOutEp),
			Len:     uint32(len(chunk)),
			Timeout: 5000, // 5 seconds
			Data:    unsafe.Pointer(&chunk[0]),
		}

		if err := ioctl(uintptr(fd), usbDevfsBulk, uintptr(unsafe.Pointer(&xfer))); err != nil {
			return fmt.Errorf("USB bulk transfer failed at offset %d: %w", offset, err)
		}
	}

	log.Printf("[RawUSB] Print job sent to %s (%d bytes, %d ciphertext bytes)",
		p.devicePath, len(escpos), len(ciphertext))
	return nil
}

// IsAvailable returns whether the raw USB printer is still accessible
func (p *RawUSBPrinter) IsAvailable() bool {
	if !p.available {
		return false
	}
	if _, err := os.Stat(p.devicePath); os.IsNotExist(err) {
		// Device was re-enumerated — try to find it again
		p.updateDevicePath()
		return p.available
	}
	return true
}

// SetAvailable sets the availability status
func (p *RawUSBPrinter) SetAvailable(available bool) {
	p.available = available
}

// HealthCheck returns health status for the raw USB printer
func (p *RawUSBPrinter) HealthCheck() map[string]interface{} {
	status := make(map[string]interface{})
	status["type"] = "rawusb"
	status["device_path"] = p.devicePath
	status["endpoint"] = fmt.Sprintf("0x%02x", p.bulkOutEp)
	status["available"] = p.IsAvailable()
	status["vid_pid"] = fmt.Sprintf("%s:%s", readSysfsFile(p.sysfsPath+"idVendor"), readSysfsFile(p.sysfsPath+"idProduct"))

	if name := readSysfsFile(p.sysfsPath + "product"); name != "" {
		status["model"] = name
	}

	return status
}

// GetDevicePath returns the USB device path
func (p *RawUSBPrinter) GetDevicePath() string {
	return p.devicePath
}

// updateDevicePath tries to re-find the device after USB re-enumeration
func (p *RawUSBPrinter) updateDevicePath() {
	vid := readSysfsFile(p.sysfsPath + "idVendor")
	pid := readSysfsFile(p.sysfsPath + "idProduct")
	if vid == "" || pid == "" {
		p.available = false
		return
	}

	newPath, newBus, newDev, _, _, err := scanUSBDevice(vid, pid)
	if err != nil {
		p.available = false
		return
	}

	p.devicePath = fmt.Sprintf("/dev/bus/usb/%03d/%03d", newBus, newDev)
	p.sysfsPath = newPath
	p.available = true
}

// ──────────────────────────────────────────────
// USB sysfs scanning
// ──────────────────────────────────────────────

// scanUSBDevice scans /sys/bus/usb/devices/ for a device matching VID:PID,
// then finds its Bulk OUT endpoint. Returns sysfs path, bus/dev numbers,
// endpoint address, and interface number.
func scanUSBDevice(vid, pid string) (sysfsPath string, busNum, devNum int, bulkOutEp uint8, ifaceNum int, err error) {
	entries, err := os.ReadDir("/sys/bus/usb/devices/")
	if err != nil {
		return "", 0, 0, 0, 0, fmt.Errorf("cannot read /sys/bus/usb/devices: %w", err)
	}

	for _, entry := range entries {
		devPath := "/sys/bus/usb/devices/" + entry.Name() + "/"

		// sysfs entries are symlinks; IsDir() returns false for them.
		// Validate by reading idVendor instead.
		devVid := readSysfsFile(devPath + "idVendor")
		devPid := readSysfsFile(devPath + "idProduct")
		if devVid == "" || devPid == "" {
			continue
		}
		if devVid != vid || devPid != pid {
			continue
		}

		bus, err1 := strconv.Atoi(readSysfsFile(devPath + "busnum"))
		dev, err2 := strconv.Atoi(readSysfsFile(devPath + "devnum"))
		if err1 != nil || err2 != nil {
			continue
		}

		// Scan interfaces for Bulk OUT endpoint.
		// No trailing "/" — filepath.Glob returns 0 matches for entries
		// inside symlinked directories when the pattern ends with "/".
		matches, err := filepath.Glob(devPath + "*:*.*")
		if err != nil {
			continue
		}

		for _, intfPath := range matches {
			intfPath += "/"

			iclass := readSysfsFile(intfPath + "bInterfaceClass")
			if iclass != "07" {
				continue
			}

			ifaceStr := readSysfsFile(intfPath + "bInterfaceNumber")
			ifaceN, _ := strconv.Atoi(ifaceStr)

			epMatches, err := filepath.Glob(intfPath + "ep_*")
			if err != nil {
				continue
			}

			for _, epPath := range epMatches {
				epType := readSysfsFile(epPath + "/type")
				epDir := readSysfsFile(epPath + "/direction")
				epAddr := readSysfsFile(epPath + "/bEndpointAddress")

				if epType == "Bulk" && epDir == "out" {
					addr, _ := strconv.ParseUint(epAddr, 16, 8)
					log.Printf("[RawUSB] Found %s:%s at %s (bus %d dev %d), Bulk OUT ep 0x%02x, iface %d",
						vid, pid, devPath, bus, dev, addr, ifaceN)
					return devPath, bus, dev, uint8(addr), ifaceN, nil
				}
			}
		}
	}

	return "", 0, 0, 0, 0, fmt.Errorf("no USB device with VID:PID %s:%s and Bulk OUT endpoint found", vid, pid)
}

// readSysfsFile reads a small file from sysfs, trimming whitespace
func readSysfsFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ioctl performs a Linux ioctl system call
func ioctl(fd, cmd, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, arg)
	if errno != 0 {
		return fmt.Errorf("ioctl 0x%x: %w", cmd, errno)
	}
	return nil
}

// DetectRawUSBPrinter tries to find a known thermal printer on the USB bus
// and returns a RawUSBPrinter if one is found. Returns nil if no supported
// printer is detected.
func DetectRawUSBPrinter() (*RawUSBPrinter, error) {
	for _, kp := range knownPrinters {
		p, err := NewRawUSBPrinter(kp.vendorID, kp.productID)
		if err == nil {
			log.Printf("[RawUSB] Detected %s (%s:%s) at %s",
				kp.name, kp.vendorID, kp.productID, p.devicePath)
			return p, nil
		}
	}
	return nil, fmt.Errorf("no supported USB thermal printer found on USB bus")
}
