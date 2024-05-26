package config

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/codec"
	"github.com/allape/openkvm/kvm/codec/tight"
)

func VideoCodecFromConfig(conf config.Config) (codec.Codec, error) {
	return &tight.JPEGEncoder{Quality: conf.Video.Quality}, nil
}
