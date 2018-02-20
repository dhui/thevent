package thevent_test

import (
	"context"
	"fmt"
	"time"
)

import "github.com/dhui/thevent"

// Song and Playlist data structures and functions
type song struct {
	Name     string
	Artist   string
	Duration time.Duration
}
type playlist struct {
	Name  string
	Songs []song
}
type songPlaylist struct {
	Song     song
	Playlist playlist
}
type playlistSwapSongs struct {
	Playlist playlist
	IdxA     int
	IdxB     int
}

func newPlaylist(ctx context.Context, name string, songs ...song) (*playlist, error) {
	p := playlist{Name: name, Songs: songs}
	if err := playlistCreatedEvent.Dispatch(ctx, p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *playlist) queueSong(ctx context.Context, song song) error {
	p.Songs = append(p.Songs, song)
	return queuedSongEvent.Dispatch(ctx, songPlaylist{Song: song, Playlist: *p})
}

func (p *playlist) swapSongs(ctx context.Context, idxA, idxB int) error {
	p.Songs[idxA], p.Songs[idxB] = p.Songs[idxB], p.Songs[idxA]
	return swappedSongsEvent.Dispatch(ctx, playlistSwapSongs{Playlist: *p, IdxA: idxA, IdxB: idxB})
}

// Event Handlers
func playlistCreatedHandler(ctx context.Context, p playlist) error {
	fmt.Printf("Created playlist %q with songs: %v\n", p.Name, p.Songs)
	return nil
}
func queuedSongHandler(ctx context.Context, sp songPlaylist) error {
	fmt.Printf("Queued song %q into playlist %q\n", sp.Song.Name, sp.Playlist.Name)
	return nil
}
func swappedSongHandler(ctx context.Context, pss playlistSwapSongs) error {
	fmt.Printf("Swapped songs %q and %q in playlist %q\n", pss.Playlist.Songs[pss.IdxB].Name,
		pss.Playlist.Songs[pss.IdxA].Name, pss.Playlist.Name)
	return nil
}

// Helpers
func mustD(d time.Duration, err error) time.Duration {
	if err != nil {
		panic(err)
	}
	return d
}

var (
	// Parent event
	playlistEvent = thevent.Must(thevent.New(playlist{}))
	// Child events
	playlistCreatedEvent = thevent.Must(playlistEvent.New(playlist{}, "", playlistCreatedHandler))
	queuedSongEvent      = thevent.Must(playlistEvent.New(songPlaylist{}, "Playlist", queuedSongHandler))
	swappedSongsEvent    = thevent.Must(playlistEvent.New(playlistSwapSongs{}, "Playlist", swappedSongHandler))
)

func Example() {
	ctx := context.Background()
	p, err := newPlaylist(ctx, "Best of Jimi")
	if err != nil {
		fmt.Println(err)
	}

	if err := p.queueSong(ctx, song{Name: "Purple Haze", Artist: "Jimi Hendrix",
		Duration: mustD(time.ParseDuration("2m46s"))}); err != nil {
		fmt.Println(err)
	}

	if err := p.queueSong(ctx, song{Name: "Foxy Lady", Artist: "Jimi Hendrix",
		Duration: mustD(time.ParseDuration("3m19s"))}); err != nil {
		fmt.Println(err)
	}

	if err := p.swapSongs(ctx, 0, 1); err != nil {
		fmt.Println(err)
	}

	playlistHandler := func(ctx context.Context, p playlist) error {
		fmt.Printf("Top-level playlist event got playlist: %q\n", p.Name)
		return nil
	}
	// Handlers may be added later
	if err := playlistEvent.AddHandlers(playlistHandler); err != nil {
		fmt.Println(err)
	}
	// Dispatching the parent event will also dispatch the children. In this case, doing so is nonsensical.
	if err := playlistEvent.Dispatch(ctx, *p); err != nil {
		fmt.Println(err)
	}

	// Output:
	// Created playlist "Best of Jimi" with songs: []
	// Queued song "Purple Haze" into playlist "Best of Jimi"
	// Queued song "Foxy Lady" into playlist "Best of Jimi"
	// Swapped songs "Purple Haze" and "Foxy Lady" in playlist "Best of Jimi"
	// Top-level playlist event got playlist: "Best of Jimi"
	// Created playlist "Best of Jimi" with songs: [{Foxy Lady Jimi Hendrix 3m19s} {Purple Haze Jimi Hendrix 2m46s}]
	// Queued song "" into playlist "Best of Jimi"
	// Swapped songs "Foxy Lady" and "Foxy Lady" in playlist "Best of Jimi"
}
