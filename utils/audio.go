// special thanks: https://github.com/bwmarrin/dgvoice/blob/master/dgvoice.go
// this is not a chat room.

package utils

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

var (
	speakers     map[uint32]*gopus.Decoder
	opusEncoder  *gopus.Encoder
	ErrPcmClosed = errors.New("err: PCM Channel closed")
	youtubeRegex = regexp.MustCompile(`(?i)^((?:https?:)?\/\/)?((?:www|m(usic)?)\.)?((?:youtube(?:-nocookie)?\.com|youtu.be))(\/.*)?$`)
)

func safeKill(proc *os.Process) {
	if proc != nil {
		_ = proc.Kill()
	}
}

func SendPCM(v *discordgo.VoiceConnection, pcm <-chan []int16) error {
	opusEncoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		return err
	}

	for recv := range pcm {
		opus, err := opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			return err
		}

		if !v.Ready || v.OpusSend == nil {
			return nil
		}
		v.OpusSend <- opus
	}
	return ErrPcmClosed
}

func PlayAudioFile(v *discordgo.VoiceConnection, source string, stop <-chan bool, playing *bool) error {
	var (
		ytdlp    *exec.Cmd
		ytdlpOut io.ReadCloser
		err      error
	)

	isUrl := youtubeRegex.MatchString(source)

	if isUrl {
		ytdlp = exec.Command("yt-dlp", "--no-part", "--downloader", "ffmpeg",
			"--buffer-size", "16K", "--limit-rate", "50K", "-o", "-", "-f", "bestaudio", source)
		source = "-"
		ytdlpOut, err = ytdlp.StdoutPipe()
		if err != nil {
			return err
		}
		if err := ytdlp.Start(); err != nil {
			return err
		}
		go func() {
			if err := ytdlp.Wait(); err != nil && !errors.Is(err, exec.ErrProcessDone) {
				fmt.Println("yt-dlp process exited with error:", err)
			}
		}()
		defer safeKill(ytdlp.Process)
	}

	ffmpeg := exec.Command("ffmpeg", "-i", source, "-f", "s16le", "-ar",
		strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")

	if isUrl {
		ffmpeg.Stdin = ytdlpOut
	}

	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		return err
	}
	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	if err = ffmpeg.Start(); err != nil {
		return err
	}
	defer safeKill(ffmpeg.Process)

	go func() {
		select {
		case <-stop:
			safeKill(ffmpeg.Process)
			if isUrl {
				safeKill(ytdlp.Process)
			}
		default:
			return
		}
	}()

	if err = v.Speaking(true); err != nil {
		return err
	}
	defer v.Speaking(false)

	send := make(chan []int16, 2)
	defer close(send)

	closeChan := make(chan bool)
	go func() {
		err := SendPCM(v, send)
		if err != nil && !errors.Is(err, ErrPcmClosed) {
			fmt.Println("SendPCM error:", err)
		}
		closeChan <- true
	}()

	for {
		if !*playing {
			time.Sleep(1 * time.Second)
			continue
		}
		audiobuf := make([]int16, frameSize*channels)
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case send <- audiobuf:
		case <-closeChan:
			return nil
		}
	}
}
