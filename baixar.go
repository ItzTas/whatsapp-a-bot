package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const (
	mediaMaxMB       = 60
	mediaMaxBytes    = mediaMaxMB << 20
	mediaMaxItems    = 10
	mediaMaxDuration = 600
	browserUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0"
)

var (
	linkPattern      = regexp.MustCompile(`https?://[^\s<>"]+`)
	errNoMedia       = errors.New("nenhuma mídia encontrada no link")
	errMediaTooLarge = fmt.Errorf("mídia passa de %d MB", mediaMaxMB)
)

type outgoing struct {
	data      []byte
	mimetype  string
	thumbnail []byte
	video     bool
	gif       bool
}

func (b *bot) baixar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	quoted := evt.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	link := cmp.Or(linkPattern.FindString(messageText(evt.Message)), linkPattern.FindString(messageText(quoted)))
	if link == "" {
		return b.sendText(ctx, chat, "Manda o link junto, tipo !baixar https://... Ou responde uma mensagem que tenha o link com !baixar")
	}
	err := b.withMediaJob(ctx, evt, func(ctx context.Context) error { return b.downloadAndSend(ctx, evt, link) })
	if err == nil {
		return nil
	}
	b.client.Log.Warnf("baixar %s: %v", link, err)
	return b.sendText(ctx, chat, fmt.Sprintf("Não consegui baixar esse link 😕 Talvez o post seja privado, o site não funcione com o bot ou o arquivo passe de %d MB ou de %d minutos", mediaMaxMB, mediaMaxDuration/60))
}

func (b *bot) answerLinks(ctx context.Context, evt *events.Message) error {
	links := linkPattern.FindAllString(messageText(evt.Message), -1)
	i := slices.IndexFunc(links, autoDownloadable)
	if i < 0 {
		return nil
	}
	on, err := b.modes.enabled(ctx, evt.Info.Chat.ToNonAD(), modeUtilities)
	if err != nil || !on {
		return err
	}
	return b.withMediaJob(ctx, evt, func(ctx context.Context) error { return b.downloadAndSend(ctx, evt, links[i]) })
}

func autoDownloadable(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	mediaExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".webm", ".mov"}
	if slices.Contains(mediaExtensions, strings.ToLower(path.Ext(u.Path))) {
		return true
	}
	sites := []string{
		"instagram.com", "tiktok.com", "x.com", "twitter.com", "youtube.com", "youtu.be",
		"reddit.com", "redd.it", "facebook.com", "fb.watch", "threads.net", "threads.com",
		"pinterest.com", "pin.it", "kwai.com", "tumblr.com", "imgur.com", "giphy.com", "tenor.com",
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	return slices.ContainsFunc(sites, func(site string) bool { return host == site || strings.HasSuffix(host, "."+site) })
}

func (b *bot) downloadAndSend(ctx context.Context, evt *events.Message, link string) (err error) {
	dir, err := os.MkdirTemp("", "baixar-*")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()

	files, err := downloadMedia(ctx, dir, link)
	if err != nil {
		return err
	}
	reply := replyTo(evt)
	sent := 0
	for _, file := range files[:min(len(files), mediaMaxItems)] {
		m, ok, err := prepareMedia(ctx, file)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := b.sendMedia(ctx, evt.Info.Chat, m, reply); err != nil {
			return err
		}
		sent++
	}
	if sent == 0 {
		return errNoMedia
	}
	return nil
}

func downloadMedia(ctx context.Context, dir, link string) ([]string, error) {
	file, directErr := fetchDirect(ctx, dir, link)
	if file != "" {
		return []string{file}, nil
	}
	errs := []error{directErr}
	for i, download := range []func(ctx context.Context, dir, link string) ([]string, error){ytDlp, galleryDl} {
		files, err := download(ctx, filepath.Join(dir, strconv.Itoa(i)), link)
		if len(files) > 0 {
			return files, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(append(errs, errNoMedia)...)
}

func fetchDirect(ctx context.Context, dir, link string) (file string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	family, _, _ := strings.Cut(resp.Header.Get("Content-Type"), "/")
	if resp.StatusCode != http.StatusOK || !slices.Contains([]string{"image", "video"}, family) {
		return "", nil
	}
	if resp.ContentLength > mediaMaxBytes {
		return "", errMediaTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, mediaMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > mediaMaxBytes {
		return "", errMediaTooLarge
	}
	file = filepath.Join(dir, "link"+strings.ToLower(path.Ext(req.URL.Path)))
	if err := os.WriteFile(file, data, 0o600); err != nil {
		return "", err
	}
	return file, nil
}

func ytDlp(ctx context.Context, dir, link string) ([]string, error) {
	return runYtDlp(ctx, dir, link, "--format-sort", "vcodec:h264,res:720,acodec:aac,ext:mp4:m4a", "--merge-output-format", "mp4")
}

func ytDlpAudio(ctx context.Context, dir, link string) ([]string, error) {
	return runYtDlp(ctx, dir, link, "--extract-audio", "--audio-format", "mp3", "--audio-quality", mp3Quality)
}

func runYtDlp(ctx context.Context, dir, link string, format ...string) ([]string, error) {
	args := slices.Concat(
		[]string{"--no-playlist", "--quiet", "--no-warnings",
			"--max-filesize", fmt.Sprintf("%dM", mediaMaxMB),
			"--match-filter", fmt.Sprintf("duration <=? %d", mediaMaxDuration)},
		format,
		[]string{"--paths", dir, "--output", "%(id)s.%(ext)s", link},
	)
	err := runTool(ctx, "yt-dlp", args...)
	files, listErr := downloadedFiles(dir)
	return files, errors.Join(err, listErr)
}

func galleryDl(ctx context.Context, dir, link string) ([]string, error) {
	err := runTool(ctx, "gallery-dl",
		"--quiet",
		"--range", fmt.Sprintf("1-%d", mediaMaxItems),
		"--filesize-max", fmt.Sprintf("%dM", mediaMaxMB),
		"--directory", dir,
		link,
	)
	files, listErr := downloadedFiles(dir)
	return files, errors.Join(err, listErr)
}

func downloadedFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		unfinished := slices.Contains([]string{".part", ".ytdl"}, filepath.Ext(e.Name()))
		if e.Type().IsRegular() && !unfinished {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

func prepareMedia(ctx context.Context, file string) (outgoing, bool, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return outgoing{}, false, err
	}
	mimetype := fileMimetype(file, data)
	exact := map[string]func(context.Context, string) (outgoing, error){
		"image/gif":  gifMedia,
		"image/jpeg": imageMedia,
		"image/png":  imageMedia,
		"video/mp4":  videoMedia,
	}
	byFamily := map[string]func(context.Context, string) (outgoing, error){
		"image": convertedImageMedia,
		"video": convertedVideoMedia,
	}
	prepare, ok := exact[mimetype]
	if !ok {
		family, _, _ := strings.Cut(mimetype, "/")
		prepare, ok = byFamily[family]
	}
	if !ok {
		return outgoing{}, false, nil
	}
	m, err := prepare(ctx, file)
	return m, true, err
}

func fileMimetype(file string, data []byte) string {
	sniffed := http.DetectContentType(data)
	if sniffed != "application/octet-stream" {
		return sniffed
	}
	byExtension := map[string]string{
		".mp4": "video/mp4", ".m4v": "video/mp4", ".mov": "video/quicktime",
		".mkv": "video/x-matroska", ".3gp": "video/3gpp",
	}
	return byExtension[strings.ToLower(filepath.Ext(file))]
}

func imageMedia(_ context.Context, file string) (outgoing, error) {
	data, err := os.ReadFile(file)
	return outgoing{data: data, mimetype: http.DetectContentType(data)}, err
}

func convertedImageMedia(ctx context.Context, file string) (outgoing, error) {
	out := file + ".jpg"
	if err := runTool(ctx, "ffmpeg", "-y", "-loglevel", "error", "-i", file, "-frames:v", "1", out); err != nil {
		return outgoing{}, err
	}
	return imageMedia(ctx, out)
}

func gifMedia(ctx context.Context, file string) (outgoing, error) {
	out := file + ".mp4"
	if err := encodeMP4(ctx, file, out, "-an"); err != nil {
		return outgoing{}, err
	}
	m, err := videoMedia(ctx, out)
	m.gif = true
	return m, err
}

func convertedVideoMedia(ctx context.Context, file string) (outgoing, error) {
	out := file + ".mp4"
	if err := encodeMP4(ctx, file, out, "-c:a", "aac"); err != nil {
		return outgoing{}, err
	}
	return videoMedia(ctx, out)
}

func encodeMP4(ctx context.Context, in, out string, extra ...string) error {
	args := slices.Concat(
		[]string{"-y", "-loglevel", "error", "-i", in,
			"-c:v", "libx264", "-preset", "veryfast", "-crf", "26", "-pix_fmt", "yuv420p",
			"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2", "-movflags", "+faststart"},
		extra,
		[]string{out},
	)
	return runTool(ctx, "ffmpeg", args...)
}

func videoMedia(ctx context.Context, file string) (outgoing, error) {
	thumb := file + ".jpg"
	if err := runTool(ctx, "ffmpeg", "-y", "-loglevel", "error", "-i", file, "-frames:v", "1", "-vf", "scale=320:-2", thumb); err != nil {
		return outgoing{}, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return outgoing{}, err
	}
	thumbnail, err := os.ReadFile(thumb)
	return outgoing{data: data, mimetype: "video/mp4", thumbnail: thumbnail, video: true}, err
}

func (b *bot) sendMedia(ctx context.Context, chat types.JID, m outgoing, reply *waE2E.ContextInfo) error {
	if m.video {
		return b.sendVideo(ctx, chat, m, reply)
	}
	return b.sendImage(ctx, chat, m, reply)
}

func (b *bot) sendImage(ctx context.Context, chat types.JID, m outgoing, reply *waE2E.ContextInfo) error {
	up, err := b.client.Upload(ctx, m.data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("enviar imagem: %w", err)
	}
	_, err = b.client.SendMessage(ctx, chat, &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		Mimetype:      proto.String(m.mimetype),
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		ContextInfo:   reply,
	}})
	return err
}

func (b *bot) sendVideo(ctx context.Context, chat types.JID, m outgoing, reply *waE2E.ContextInfo) error {
	up, err := b.client.Upload(ctx, m.data, whatsmeow.MediaVideo)
	if err != nil {
		return fmt.Errorf("enviar vídeo: %w", err)
	}
	_, err = b.client.SendMessage(ctx, chat, &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		Mimetype:      proto.String(m.mimetype),
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		GifPlayback:   proto.Bool(m.gif),
		JPEGThumbnail: m.thumbnail,
		ContextInfo:   reply,
	}})
	return err
}
