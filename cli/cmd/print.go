package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"huawei.com/devbridge/internal/i18n"
)

func printKV(kv [][2]string) {
	maxKeyWidth := 0
	for _, pair := range kv {
		w := i18n.DisplayWidth(pair[0])
		if w > maxKeyWidth {
			maxKeyWidth = w
		}
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

func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
}

func FormatTunnelExpiration(tunnelExpiration int64) string {
	if tunnelExpiration >= 24 {
		days := tunnelExpiration / 24
		return fmt.Sprintf("%d %s", days, i18n.T(i18n.Msg.Common.Days))
	}
	return fmt.Sprintf("%d %s", tunnelExpiration, i18n.T(i18n.Msg.Common.Hours))
}

func FormatTunnelRemaining(expireAt int64) string {
	now := time.Now().Unix()
	remaining := expireAt - now
	if remaining <= 0 {
		return i18n.T(i18n.Msg.Common.Expired)
	}
	hours := float64(remaining) / 3600
	if hours >= 24 {
		days := hours / 24
		return formatRemaining(days, i18n.T(i18n.Msg.Common.Days))
	}
	return formatRemaining(hours, i18n.T(i18n.Msg.Common.Hours))
}

// formatRemaining formats a value with one decimal place, ceiling rounded.
// If the decimal part is 0, only the integer part is shown.
func formatRemaining(v float64, unit string) string {
	// Ceiling round to 1 decimal place: ceil(v * 10) / 10
	rounded := math.Ceil(v*10) / 10
	if rounded == math.Trunc(rounded) {
		return fmt.Sprintf("%d %s", int(rounded), unit)
	}
	return fmt.Sprintf("%.1f %s", rounded, unit)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatTime(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}
