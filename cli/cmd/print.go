package cmd

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"huawei.com/devbridge/internal/i18n"
)

func printKV(kv [][2]string) {
	maxKeyWidth := 0
	for _, pair := range kv {
		w := i18n.DisplayWidth(pair[0])
		maxKeyWidth = max(maxKeyWidth, w)
	}
	for _, pair := range kv {
		keyWithColon := pair[0] + ":"
		paddedKey := i18n.PadRight(keyWithColon, maxKeyWidth+5)
		fmt.Printf("%s%s\n", paddedKey, pair[1])
	}
}

func printTable(headers []string, rows [][]string) {
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = i18n.DisplayWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				w := i18n.DisplayWidth(cell)
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}
	for i, h := range headers {
		fmt.Print(i18n.PadRight(h, colWidths[i]))
		if i < len(headers)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				fmt.Print(i18n.PadRight(cell, colWidths[i]))
			}
			if i < len(row)-1 {
				fmt.Print("  ")
			}
		}
		fmt.Println()
	}
}

func formatTunnelExpiration(tunnelExpiration int64) string {
	return formatDuration(float64(tunnelExpiration))
}

func formatTunnelRemaining(expireAt int64) string {
	now := time.Now().Unix()
	remaining := expireAt - now
	if remaining <= 0 {
		return i18n.T(i18n.Msg.Common.Expired)
	}
	return formatDuration(float64(remaining) / 3600)
}

// formatDuration renders an hour count with day/hour units, rounding up
// (ceiling) to 0.1 hour before choosing the unit. Ceiling first keeps a
// 24-hour tunnel consistent between create (static: exactly 24h -> 1 day) and
// show/list (remaining: 24h minus a few seconds -> would otherwise round to
// 24 hours).
func formatDuration(hours float64) string {
	rounded := math.Ceil(hours*10) / 10
	if rounded >= 24 {
		return formatRemaining(rounded/24, i18n.T(i18n.Msg.Common.Day), i18n.T(i18n.Msg.Common.Days))
	}
	return formatRemaining(rounded, i18n.T(i18n.Msg.Common.Hour), i18n.T(i18n.Msg.Common.Hours))
}

func formatRemaining(v float64, singular, plural string) string {
	rounded := math.Ceil(v*10) / 10

	unit := plural
	if rounded == 1 {
		unit = singular
	}
	return strconv.FormatFloat(rounded, 'f', -1, 64) + " " + unit
}

var byteUnits = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}

func formatBytes(b int64) string {
	const unit = 1024.0
	f := float64(b)
	i := 0
	for f >= unit && i < len(byteUnits)-1 {
		f /= unit
		i++
	}

	f = math.Round(f*10) / 10
	return strconv.FormatFloat(f, 'f', -1, 64) + " " + byteUnits[i]
}

func formatTime(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}
