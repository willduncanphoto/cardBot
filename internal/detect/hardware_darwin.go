//go:build darwin

package detect

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation

#include <IOKit/IOKitLib.h>
#include <IOKit/storage/IOMedia.h>
#include <CoreFoundation/CoreFoundation.h>

// getDeviceInfo queries IOKit for additional device information.
// Note: USB card readers do not expose SD card CID register.
// We can only get USB mass storage device info, not raw SD card info.
static int64_t getDeviceSize(const char* bsdName) {
    io_iterator_t iterator;
    mach_port_t masterPort;
    IOMasterPort(MACH_PORT_NULL, &masterPort);
    
    CFMutableDictionaryRef matching = IOBSDNameMatching(masterPort, 0, bsdName);
    if (!matching) return -1;
    
    kern_return_t kr = IOServiceGetMatchingServices(masterPort, matching, &iterator);
    if (kr != KERN_SUCCESS) return -1;
    
    io_object_t service;
    int64_t size = -1;
    
    while ((service = IOIteratorNext(iterator))) {
        CFNumberRef sizeNum = (CFNumberRef)IORegistryEntryCreateCFProperty(
            service, CFSTR(kIOMediaSizeKey), kCFAllocatorDefault, 0);
        if (sizeNum) {
            CFNumberGetValue(sizeNum, kCFNumberSInt64Type, &size);
            CFRelease(sizeNum);
        }
        IOObjectRelease(service);
        break;
    }
    
    IOObjectRelease(iterator);
    return size;
}
*/
import "C"
import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// HardwareInfo contains device-level information about the card/reader.
// Note: On macOS with USB readers, this is limited to USB mass storage data.
// The SD card CID register (manufacturer ID, serial number, etc.) is not
// accessible through USB. Linux with direct SD slot access can read CID.
type HardwareInfo struct {
	// Device size from IOKit (may differ from filesystem size due to partitioning/formatting)
	DeviceBytes int64

	// Filesystem size (what Statfs reports)
	FilesystemBytes int64

	// Block size
	BlockSize int64

	// Device identifier (e.g., "disk4s1")
	DeviceID string

	// Volume UUID if available
	VolumeUUID string

	// Whether this is a removable device
	IsRemovable bool

	// Protocol (USB, SD Card, etc.)
	Protocol string

	// Whether we could read CID (only true on Linux with direct SD slot)
	CIDAvailable bool

	// Raw CID hex string (Linux only, when CIDAvailable is true)
	CIDHex string

	// Parsed CID fields (Linux only)
	ManufacturerID   byte   // MID
	OEMID            string // OID
	ProductName      string // PNM
	ProductRevision  byte   // PRV
	ProductSerial    uint32 // PSN
	ManufacturingDate string // MDT
}

// GetHardwareInfo attempts to retrieve hardware information for the given mount path.
// On macOS with USB readers, this returns limited info (no CID access).
func GetHardwareInfo(mountPath string) (*HardwareInfo, error) {
	info := &HardwareInfo{
		CIDAvailable: false, // USB readers don't expose CID
	}

	// Get filesystem stats
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mountPath, &stat); err != nil {
		return nil, err
	}
	info.FilesystemBytes = int64(stat.Blocks) * int64(stat.Bsize)
	info.BlockSize = int64(stat.Bsize)

	// Try to get device info from diskutil
	deviceID, err := getDeviceID(mountPath)
	if err != nil {
		return info, nil // Return what we have
	}
	info.DeviceID = deviceID

	// Query diskutil for device properties
	diskInfo, err := getDiskUtilInfo(deviceID)
	if err == nil {
		info.DeviceBytes = diskInfo.TotalSize
		info.IsRemovable = diskInfo.Removable
		info.Protocol = diskInfo.Protocol
		info.VolumeUUID = diskInfo.VolumeUUID
	}

	return info, nil
}

type diskUtilInfo struct {
	TotalSize   int64
	Removable   bool
	Protocol    string
	VolumeUUID  string
	DeviceBlockSize int64
}

func getDeviceID(mountPath string) (string, error) {
	// Use diskutil to find the device identifier
	cmd := exec.Command("diskutil", "info", mountPath)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse "Device Identifier: disk4s1"
	re := regexp.MustCompile(`Device Identifier:\s*(\S+)`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) >= 2 {
		return matches[1], nil
	}

	return "", fmt.Errorf("device identifier not found")
}

func getDiskUtilInfo(deviceID string) (*diskUtilInfo, error) {
	// Get parent disk (e.g., disk4s1 -> disk4)
	parentDisk := deviceID
	if strings.Contains(deviceID, "s") {
		parts := strings.Split(deviceID, "s")
		if len(parts) >= 2 {
			parentDisk = parts[0]
		}
	}

	// Query the physical disk for size info
	cmd := exec.Command("diskutil", "info", parentDisk)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	info := &diskUtilInfo{}
	output := string(out)

	// Parse Total Size
	// Disk Size: 512.1 GB (512110190592 Bytes)
	re := regexp.MustCompile(`Disk Size:\s*[\d.]+\s*\w+\s*\((\d+)\s*Bytes\)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		size, _ := strconv.ParseInt(matches[1], 10, 64)
		info.TotalSize = size
	}

	// Parse Removable
	if strings.Contains(output, "Removable Media:") {
		re = regexp.MustCompile(`Removable Media:\s*(\w+)`)
		matches = re.FindStringSubmatch(output)
		if len(matches) >= 2 {
			info.Removable = matches[1] == "Yes" || matches[1] == "Removable"
		}
	}

	// Parse Protocol
	re = regexp.MustCompile(`Protocol:\s*(.+)`)
	matches = re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		info.Protocol = strings.TrimSpace(matches[1])
	}

	// Parse Device Block Size
	re = regexp.MustCompile(`Device Block Size:\s*(\d+)\s*Bytes`)
	matches = re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		bs, _ := strconv.ParseInt(matches[1], 10, 64)
		info.DeviceBlockSize = bs
	}

	// Get volume UUID from child device
	cmd = exec.Command("diskutil", "info", deviceID)
	out, _ = cmd.Output()
	volumeOutput := string(out)

	re = regexp.MustCompile(`Volume UUID:\s*(\S+)`)
	matches = re.FindStringSubmatch(volumeOutput)
	if len(matches) >= 2 {
		info.VolumeUUID = matches[1]
	}

	return info, nil
}

// FormatHardwareInfo returns a formatted string with hardware details.
func FormatHardwareInfo(info *HardwareInfo) string {
	if info == nil {
		return "Hardware info unavailable"
	}

	var parts []string

	parts = append(parts, fmt.Sprintf("Device: %s", info.DeviceID))

	if info.Protocol != "" {
		parts = append(parts, fmt.Sprintf("Protocol: %s", info.Protocol))
	}

	if info.DeviceBytes > 0 {
		parts = append(parts, fmt.Sprintf("Raw Size: %s", FormatBytes(info.DeviceBytes)))
	}

	if info.FilesystemBytes > 0 {
		parts = append(parts, fmt.Sprintf("Filesystem: %s", FormatBytes(info.FilesystemBytes)))
	}

	if info.IsRemovable {
		parts = append(parts, "Removable: Yes")
	}

	if info.VolumeUUID != "" {
		parts = append(parts, fmt.Sprintf("UUID: %s", info.VolumeUUID))
	}

	if info.CIDAvailable {
		parts = append(parts, "CID: Available")
		parts = append(parts, fmt.Sprintf("Manufacturer: 0x%02X", info.ManufacturerID))
		parts = append(parts, fmt.Sprintf("OEM: %s", info.OEMID))
		parts = append(parts, fmt.Sprintf("Product: %s", info.ProductName))
		parts = append(parts, fmt.Sprintf("Revision: %d.%d", info.ProductRevision>>4, info.ProductRevision&0x0F))
		parts = append(parts, fmt.Sprintf("Serial: 0x%08X", info.ProductSerial))
		parts = append(parts, fmt.Sprintf("Mfg Date: %s", info.ManufacturingDate))
	} else {
		parts = append(parts, "CID: Not available (USB reader)")
	}

	return strings.Join(parts, "\n  ")
}
