package sound

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

var (
	initOnce  sync.Once
	mu        sync.Mutex
	isPlaying bool
	curCtrl   *beep.Ctrl
	curStream beep.StreamSeekCloser
)

func initSpeaker(sampleRate beep.SampleRate) {
	initOnce.Do(func() {
		if err := speaker.Init(sampleRate, sampleRate.N(time.Second/10)); err != nil {
			log.Println("[sound] speaker.Init failed:", err)
		} else {
			log.Println("[sound] speaker initialized")
		}
	})
}

func PlayMusic() {
	mu.Lock()
	defer mu.Unlock()

	if isPlaying {
		return
	}

	path := findAudioFile()
	if path == "" {
		log.Println("[sound] playback.mp3 not found")
		return
	}

	f, err := os.Open(path)
	if err != nil {
		log.Println("[sound] open failed:", err)
		return
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Println("[sound] decode failed:", err)
		f.Close()
		return
	}

	initSpeaker(format.SampleRate)

	if curCtrl != nil {
		speaker.Lock()
		curCtrl.Streamer = nil
		speaker.Unlock()
	}
	if curStream != nil {
		curStream.Close()
	}

	ctrl := &beep.Ctrl{Streamer: streamer, Paused: false}

	curCtrl = ctrl
	curStream = streamer
	isPlaying = true

	speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
		mu.Lock()
		defer mu.Unlock()
		isPlaying = false
		if curStream != nil {
			curStream.Close()
			curStream = nil
		}
		curCtrl = nil
	})))
}

func StopMusic() {
	mu.Lock()
	defer mu.Unlock()

	if !isPlaying {
		return
	}

	speaker.Lock()
	if curCtrl != nil {
		curCtrl.Streamer = nil
	}
	speaker.Unlock()

	if curStream != nil {
		curStream.Close()
		curStream = nil
	}
	curCtrl = nil
	isPlaying = false
}

func IsPlaying() bool {
	mu.Lock()
	defer mu.Unlock()
	return isPlaying
}

func findAudioFile() string {
	paths := []string{
		"sound/playback.mp3",
		"./sound/playback.mp3",
		"playback.mp3",
		"./playback.mp3",
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		paths = append(paths,
			filepath.Join(exeDir, "sound", "playback.mp3"),
			filepath.Join(exeDir, "playback.mp3"),
		)
	}
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}
