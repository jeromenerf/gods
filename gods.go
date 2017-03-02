// This programm collects some system information, formats it nicely and sets
// the X root windows name so it can be displayed in the dwm status bar.
//
// The strange characters in the output are used by dwm to colorize the output
// ( to , needs the http://dwm.suckless.org/patches/statuscolors patch) and
// as Icons or separators (e.g. "Ý"). If you don't use the status-18 font
// (https://github.com/schachmat/status-18), you should probably exchange them
// by something else ("CPU", "MEM", "|" for separators, …).
//
// For license information see the file LICENSE
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	bpsSign   = "B"
	kibpsSign = "kB"
	mibpsSign = "MB"

	unpluggedSign = "↓"
	pluggedSign   = "↑"

	cpuSign = "C"
	memSign = "M"

	audioSign = "♫"

	netReceivedSign    = "↓"
	netTransmittedSign = "↑"

	floatSeparator = ","
	dateSeparator  = ""
	fieldSeparator = "  -  "
)

var (
	netDevs = map[string]struct{}{
		"eth0:":  {},
		"eth1:":  {},
		"eth2:":  {},
		"wlan0:": {},
		"wlan1:": {},
		"wlan2:": {},
		"ppp0:":  {},
	}
	cores = runtime.NumCPU() // count of cores to scale cpu usage
	rxOld = 0
	txOld = 0
)

// fixed builds a fixed width string with given pre- and fitting suffix
func fixed(pre string, rate int) string {
	if rate < 0 {
		return pre + " ERR"
	}

	var spd = float32(rate)
	var suf = bpsSign // default: display as B/s

	switch {
	case spd >= (1000 * 1024 * 1024): // > 999 MiB/s
		return "" + pre + "ERR"
	case spd >= (1000 * 1024): // display as MiB/s
		spd /= (1024 * 1024)
		suf = mibpsSign
		pre = "" + pre + ""
	case spd >= 1000: // display as KiB/s
		spd /= 1024
		suf = kibpsSign
	}

	var formated = ""
	if spd >= 100 {
		formated = fmt.Sprintf("%3.0f", spd)
	} else if spd >= 10 {
		formated = fmt.Sprintf("%3.0f", spd)
	} else {
		formated = fmt.Sprintf("%2.1f", spd)
	}
	return pre + strings.Replace(formated, ".", floatSeparator, 1) + suf
}

// updateNetUse reads current transfer rates of certain network interfaces
func updateNetUse() string {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return netReceivedSign + " ERR " + netTransmittedSign + " ERR"
	}
	defer file.Close()

	var void = 0 // target for unused values
	var dev, rx, tx, rxNow, txNow = "", 0, 0, 0, 0
	var scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		_, err = fmt.Sscanf(scanner.Text(), "%s %d %d %d %d %d %d %d %d %d",
			&dev, &rx, &void, &void, &void, &void, &void, &void, &void, &tx)
		if _, ok := netDevs[dev]; ok {
			rxNow += rx
			txNow += tx
		}
	}

	defer func() { rxOld, txOld = rxNow, txNow }()
	return fmt.Sprintf("%s %s", fixed(netReceivedSign, rxNow-rxOld), fixed(netTransmittedSign, txNow-txOld))
}

// colored surrounds the percentage with color escapes if it is >= 70
func colored(icon string, percentage int) string {
	if percentage >= 100 {
		return fmt.Sprintf("%s%3d", icon, percentage)
	} else if percentage >= 70 {
		return fmt.Sprintf("%s%3d", icon, percentage)
	}
	return fmt.Sprintf("%s%3d", icon, percentage)
}

// updatePower reads the current battery and power plug status
func updatePower() string {
	icon := pluggedSign
	lvl, err := exec.Command("sh", "-c", `acpi -b | awk -F '[ ,]' -vORS='' '{print $5, $7}'`).Output()
	if err != nil {
		log.Println(err)
	}
	if len(lvl) > 4 {
		icon = unpluggedSign
	}
	return fmt.Sprintf("%s%s", icon, lvl)
}

// updateDateTime returns the current datetime
func updateDateTime() string {
	return time.Now().Local().Format("Mon 02 Jan 2006" + dateSeparator + " 15:04")
}

//updateAudioVolume returns the current audio mute status and Master volume
func updateAudioVolume() string {
	vol, err := exec.Command("sh", "-c", `amixer sget Master | awk -vORS='' '/Left:/ {print($5)}' | tr -d '[]'`).Output()
	if err != nil {
		log.Println(err)
	}
	_, err = exec.Command("sh", "-c", `amixer sget Master | grep -c '\[on\]'`).Output()
	if err != nil {
		vol = []byte("[mute]")
	}
	return audioSign + string(vol)
}

// main updates the dwm statusbar every second
func main() {
	for {
		var status = []string{
			updateNetUse(),
			//updateCPUUse(),
			//updateMemUse(),
			updatePower(),
			updateAudioVolume(),
			updateDateTime(),
		}
		exec.Command("xsetroot", "-name", strings.Join(status, fieldSeparator)).Run()
		fmt.Println(strings.Join(status, fieldSeparator))

		// sleep until beginning of next second
		time.Sleep(5 * time.Second)
	}
}
