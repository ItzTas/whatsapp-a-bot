package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types/events"
)

const (
	weatherGeocodingEndpoint = "https://geocoding-api.open-meteo.com/v1/search"
	weatherForecastEndpoint  = "https://api.open-meteo.com/v1/forecast"
	weatherTimeout           = 15 * time.Second
	weatherCityMax           = 80
	weatherForecastDays      = 4
	publicJSONMaxBytes       = 1 << 20
)

var (
	errWeatherCityNotFound = errors.New("cidade não encontrada")
	errPublicNotFound      = errors.New("não encontrado")
)

type weatherPlace struct {
	Name      string  `json:"name"`
	Admin     string  `json:"admin1"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type weatherForecast struct {
	Current struct {
		Temperature   float64 `json:"temperature_2m"`
		FeelsLike     float64 `json:"apparent_temperature"`
		Humidity      float64 `json:"relative_humidity_2m"`
		Precipitation float64 `json:"precipitation"`
		Code          int     `json:"weather_code"`
		Wind          float64 `json:"wind_speed_10m"`
		IsDay         int     `json:"is_day"`
	} `json:"current"`
	Daily struct {
		Time       []string  `json:"time"`
		Code       []int     `json:"weather_code"`
		Max        []float64 `json:"temperature_2m_max"`
		Min        []float64 `json:"temperature_2m_min"`
		RainChance []float64 `json:"precipitation_probability_max"`
	} `json:"daily"`
}

func (b *bot) clima(ctx context.Context, evt *events.Message) error {
	city := textAfterFields(messageText(evt.Message), 1)
	if city == "" {
		return b.replyTo(ctx, evt, "🌦️ Use assim: !clima <cidade>, tipo !clima São Paulo ou !clima Recife. Eu mando como tá o tempo agora e a previsão dos próximos dias")
	}
	if utf8.RuneCountInString(city) > weatherCityMax {
		return b.replyTo(ctx, evt, fmt.Sprintf("Nome de cidade grande demais, o máximo é %d letras", weatherCityMax))
	}

	stop := b.typing(ctx, evt.Info.Chat)
	place, forecast, err := fetchWeather(ctx, city)
	stop()
	if errors.Is(err, errWeatherCityNotFound) {
		return b.replyTo(ctx, evt, fmt.Sprintf("🗺️ Não achei nenhuma cidade chamada \"%s\". Confere o nome e tenta de novo", city))
	}
	if err != nil {
		b.client.Log.Warnf("!clima %q: %v", city, err)
		return b.replyTo(ctx, evt, "Não consegui ver a previsão agora 😕 Tenta de novo daqui a pouco")
	}
	return b.replyTo(ctx, evt, weatherReport(place, forecast))
}

func fetchWeather(ctx context.Context, city string) (weatherPlace, weatherForecast, error) {
	ctx, cancel := context.WithTimeout(ctx, weatherTimeout)
	defer cancel()
	place, err := findWeatherPlace(ctx, city)
	if err != nil {
		return weatherPlace{}, weatherForecast{}, err
	}
	query := url.Values{
		"latitude":      {fmt.Sprint(place.Latitude)},
		"longitude":     {fmt.Sprint(place.Longitude)},
		"current":       {"temperature_2m,apparent_temperature,relative_humidity_2m,precipitation,weather_code,wind_speed_10m,is_day"},
		"daily":         {"weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max"},
		"timezone":      {"auto"},
		"forecast_days": {fmt.Sprint(weatherForecastDays)},
	}
	var forecast weatherForecast
	err = fetchPublicJSON(ctx, weatherForecastEndpoint+"?"+query.Encode(), &forecast)
	return place, forecast, err
}

func findWeatherPlace(ctx context.Context, city string) (weatherPlace, error) {
	query := url.Values{"name": {city}, "language": {"pt"}, "count": {"1"}}
	var found struct {
		Results []weatherPlace `json:"results"`
	}
	if err := fetchPublicJSON(ctx, weatherGeocodingEndpoint+"?"+query.Encode(), &found); err != nil {
		return weatherPlace{}, err
	}
	if len(found.Results) == 0 {
		return weatherPlace{}, errWeatherCityNotFound
	}
	return found.Results[0], nil
}

func weatherReport(place weatherPlace, f weatherForecast) string {
	now := f.Current
	emoji, description := weatherCondition(now.Code, now.IsDay == 1)
	lines := []string{
		"📍 *" + place.label() + "*",
		"",
		fmt.Sprintf("%s *Agora:* %s, %s", emoji, weatherCelsius(now.Temperature), description),
		"🤒 Sensação de " + weatherCelsius(now.FeelsLike),
		fmt.Sprintf("💧 Umidade de %.0f%% · 🌧️ Chuva: %s mm", now.Humidity, weatherDecimal(now.Precipitation, 1)),
		fmt.Sprintf("💨 Vento de %.0f km/h", now.Wind),
		"",
		"🗓️ *Próximos dias*",
	}
	daily := f.Daily
	for i, day := range daily.Time {
		if i >= len(daily.Code) || i >= len(daily.Max) || i >= len(daily.Min) || i >= len(daily.RainChance) {
			break
		}
		emoji, description := weatherCondition(daily.Code[i], true)
		lines = append(lines, fmt.Sprintf("%s *%s:* %s a %s, %s (%.0f%% de chance de chuva)",
			emoji, weatherDayLabel(day, i), weatherCelsius(daily.Min[i]), weatherCelsius(daily.Max[i]), description, daily.RainChance[i]))
	}
	return strings.Join(lines, "\n")
}

func (p weatherPlace) label() string {
	parts := []string{p.Name}
	for _, part := range []string{p.Admin, p.Country} {
		if part != "" && part != parts[len(parts)-1] {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

func weatherDayLabel(day string, index int) string {
	names := map[int]string{0: "Hoje", 1: "Amanhã"}
	if name, ok := names[index]; ok {
		return name
	}
	date, err := time.Parse(time.DateOnly, day)
	if err != nil {
		return day
	}
	weekdays := []string{"Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"}
	return fmt.Sprintf("%s %s", weekdays[date.Weekday()], date.Format("02/01"))
}

func weatherCondition(code int, isDay bool) (emoji, description string) {
	conditions := map[int][2]string{
		0:  {"☀️", "céu limpo"},
		1:  {"🌤️", "quase sem nuvens"},
		2:  {"⛅", "parcialmente nublado"},
		3:  {"☁️", "nublado"},
		45: {"🌫️", "neblina"},
		48: {"🌫️", "neblina com geada"},
		51: {"🌦️", "garoa fraca"},
		53: {"🌦️", "garoa"},
		55: {"🌧️", "garoa forte"},
		56: {"🌧️", "garoa congelante"},
		57: {"🌧️", "garoa congelante forte"},
		61: {"🌦️", "chuva fraca"},
		63: {"🌧️", "chuva"},
		65: {"🌧️", "chuva forte"},
		66: {"🌧️", "chuva congelante"},
		67: {"🌧️", "chuva congelante forte"},
		71: {"🌨️", "neve fraca"},
		73: {"🌨️", "neve"},
		75: {"❄️", "neve forte"},
		77: {"🌨️", "grãos de neve"},
		80: {"🌦️", "pancadas de chuva fracas"},
		81: {"🌧️", "pancadas de chuva"},
		82: {"⛈️", "pancadas de chuva fortes"},
		85: {"🌨️", "pancadas de neve"},
		86: {"❄️", "pancadas de neve fortes"},
		95: {"⛈️", "trovoada"},
		96: {"⛈️", "trovoada com granizo"},
		99: {"⛈️", "trovoada com granizo forte"},
	}
	condition, ok := conditions[code]
	if !ok {
		return "🌡️", "tempo indefinido"
	}
	nightEmojis := map[int]string{0: "🌙", 1: "🌙"}
	if night, ok := nightEmojis[code]; ok && !isDay {
		return night, condition[1]
	}
	return condition[0], condition[1]
}

func weatherCelsius(value float64) string {
	return fmt.Sprintf("%.0f°C", math.Round(value)+0)
}

func weatherDecimal(value float64, decimals int) string {
	return strings.Replace(fmt.Sprintf("%.*f", decimals, value), ".", ",", 1)
}

func fetchPublicJSON(ctx context.Context, link string, out any) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if resp.StatusCode == http.StatusNotFound {
		return errPublicNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, publicJSONMaxBytes)).Decode(out); err != nil {
		return fmt.Errorf("ler resposta: %w", err)
	}
	return nil
}
