package thevent_test

import (
	"context"
	"fmt"
	"sort"
	"time"
)

import "github.com/dhui/thevent"

// Queue output due to how event handlers and children are stored
// See: https://github.com/dhui/thevent/issues/3
type printQueueType struct {
	queue []string
}

func (q *printQueueType) Append(s string) {
	q.queue = append(q.queue, s)
}

func (q *printQueueType) Clear() {
	q.queue = []string{}
}

func (q *printQueueType) Print() {
	for _, s := range q.queue {
		fmt.Println(s)
	}
}

func (q *printQueueType) Sort() {
	sort.Strings(q.queue)
}

var printQueue = printQueueType{}

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
	printQueue.Append(fmt.Sprintf("Created playlist %q with songs: %v", p.Name, p.Songs))
	return nil
}
func queuedSongHandler(ctx context.Context, sp songPlaylist) error {
	printQueue.Append(fmt.Sprintf("Queued song %q into playlist %q", sp.Song.Name, sp.Playlist.Name))
	return nil
}
func swappedSongHandler(ctx context.Context, pss playlistSwapSongs) error {
	printQueue.Append(fmt.Sprintf("Swapped songs %q and %q in playlist %q",
		pss.Playlist.Songs[pss.IdxB].Name, pss.Playlist.Songs[pss.IdxA].Name, pss.Playlist.Name))
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

	// Don't need to sort output since none of the dispatched events have multiple children or handlers
	printQueue.Print()
	printQueue.Clear()

	playlistHandler := func(ctx context.Context, p playlist) error {
		printQueue.Append(fmt.Sprintf("Top-level playlist event got playlist: %q", p.Name))
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

	// Sort output since the top-level event has multiple children
	printQueue.Sort()
	printQueue.Print()

	// Output:
	// Created playlist "Best of Jimi" with songs: []
	// Queued song "Purple Haze" into playlist "Best of Jimi"
	// Queued song "Foxy Lady" into playlist "Best of Jimi"
	// Swapped songs "Purple Haze" and "Foxy Lady" in playlist "Best of Jimi"
	// Created playlist "Best of Jimi" with songs: [{Foxy Lady Jimi Hendrix 3m19s} {Purple Haze Jimi Hendrix 2m46s}]
	// Queued song "" into playlist "Best of Jimi"
	// Swapped songs "Foxy Lady" and "Foxy Lady" in playlist "Best of Jimi"
	// Top-level playlist event got playlist: "Best of Jimi"
}
