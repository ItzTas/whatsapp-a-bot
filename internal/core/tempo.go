package core

import (
	"fmt"
	"time"
)

func FormatWait(d time.Duration) string {
	d = d.Round(time.Minute)
	hours, minutes := int(d.Hours()), int(d.Minutes())%60
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh %dmin", hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dmin", max(minutes, 1))
}

func Countdown(deadline time.Time) string {
	left := max(time.Until(deadline).Round(time.Second), time.Second)
	minutes, seconds := int(left.Minutes()), int(left.Seconds())%60
	if minutes == 0 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds == 0 {
		return fmt.Sprintf("%dmin", minutes)
	}
	return fmt.Sprintf("%dmin %ds", minutes, seconds)
}

var (
	BrazilTime = time.FixedZone("BRT", -3*60*60)
)

func BrazilianDate(t time.Time) string {
	months := []string{
		"janeiro", "fevereiro", "março", "abril", "maio", "junho",
		"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
	}
	local := t.In(BrazilTime)
	date := fmt.Sprintf("%d de %s", local.Day(), months[local.Month()-1])
	if local.Year() == time.Now().In(BrazilTime).Year() {
		return date
	}
	return fmt.Sprintf("%s de %d", date, local.Year())
}

const (
	ReminderTick = time.Minute
)

var (
	ReminderZone = time.FixedZone("BRT", -3*60*60)
)

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return CountLabel(int(d.Seconds()), "segundo", "segundos")
	}
	if d < time.Hour {
		return CountLabel(int(d.Minutes()), "minuto", "minutos")
	}
	hours := CountLabel(int(d.Hours()), "hora", "horas")
	minutes := int(d.Minutes()) % 60
	if minutes == 0 {
		return hours
	}
	return hours + " e " + CountLabel(minutes, "minuto", "minutos")
}
