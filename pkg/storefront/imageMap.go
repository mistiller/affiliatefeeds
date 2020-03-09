package storefront

import (
	"hash"
	"hash/fnv"
)

type Image struct {
	sourceLink string
	localLink  string
}

type ImageMap struct {
	m map[uint64]*Image
}

func NewImageMap() (i *ImageMap, err error) {
	return &ImageMap{
		m: make(map[uint64]*Image),
	}, nil
}

func (i *ImageMap) Get(sourceLink string) (localLink string, err error) {
	var (
		exists bool
		h      hash.Hash64
	)

	h = fnv.New64a()
	h.Write([]byte(sourceLink))

	_, exists = i.m[h.Sum64()]
	if exists {
		return i.m[h.Sum64()].localLink, nil
	}

	link, err := dummyHandler(sourceLink)
	if err != nil {
		return i.m[h.Sum64()].localLink, err
	}

	i.m[h.Sum64()] = &Image{
		sourceLink: sourceLink,
		localLink:  link,
	}

	return link, nil
}

func dummyHandler(sourceLink string) (localLink string, err error) {
	// Example A:
	// + Download Image
	// + Resize Image
	// + Clean up

	// Example B:
	// + Search Filesystem for Image File
	// + Return Path

	localLink = sourceLink
	return localLink, nil
}
