package sound

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

var errEmptyPCM = errors.New("sound: empty pcm buffer")

type ebitenOutput struct {
	ctx *audio.Context
}

func (o ebitenOutput) loop(pcm []byte, walk phraseWalk) (voice, error) {
	if len(pcm) == 0 {
		return nil, errEmptyPCM
	}
	p, err := o.ctx.NewPlayer(newJukebox(pcm, walk))
	if err != nil {
		return nil, fmt.Errorf("sound: music player: %w", err)
	}
	return p, nil
}

func (o ebitenOutput) oneShot(pcm []byte) (voice, error) {
	if len(pcm) == 0 {
		return nil, errEmptyPCM
	}
	p, err := o.ctx.NewPlayer(bytes.NewReader(pcm))
	if err != nil {
		return nil, fmt.Errorf("sound: effect player: %w", err)
	}
	return p, nil
}
