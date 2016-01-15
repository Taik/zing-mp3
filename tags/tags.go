package tags

import (
	"os"

	"github.com/mikkyang/id3-go"
)

// UpdateMP3Tags updates the os.File with the MP3 data (artist and title).
func UpdateMP3Tags(fd *os.File, artist, title string) error {
	mp3, err := id3.Open(fd.Name())
	if err != nil {
		return err
	}
	defer mp3.Close()

	// TODO: Add lyric frames
	mp3.SetArtist(artist)
	mp3.SetTitle(title)
	mp3.SetAlbum("")
	mp3.SetGenre("")
	mp3.SetYear("")

	return nil
}
