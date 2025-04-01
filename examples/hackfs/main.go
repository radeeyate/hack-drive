package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/SlyMarbo/rss"
)

const fileSuffix = ".txt"
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.TrimSpace(name)
	return name
}

type RssFile struct {
	fs.Inode

	content []byte
}

var _ = (fs.NodeOpener)((*RssFile)(nil))
var _ = (fs.NodeGetattrer)((*RssFile)(nil))
var _ = (fs.FileReader)((*RssFile)(nil))


func (f *RssFile) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	end := int(off) + len(dest)
	if end > len(f.content) {
		end = len(f.content)
	}

	if off >= int64(len(f.content)) {
		return fuse.ReadResultData([]byte{}), fs.OK
	}

	if off < 0 {
		return nil, syscall.EINVAL
	}

	return fuse.ReadResultData(f.content[off:end]), fs.OK

}

func (f *RssFile) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	return f, 0, fs.OK
}

func (f *RssFile) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0444
	out.Size = uint64(len(f.content))
	now := time.Now()
	out.SetTimes(&now, &now, &now)
	return fs.OK
}

type RssRoot struct {
	fs.Inode 

	mu       sync.Mutex
	items    map[string]*rss.Item
	feedUrl  string
	feedName string
}


var _ = (fs.NodeOnAdder)((*RssRoot)(nil))
var _ = (fs.NodeReaddirer)((*RssRoot)(nil))
var _ = (fs.NodeLookuper)((*RssRoot)(nil))
var _ = (fs.NodeGetattrer)((*RssRoot)(nil))

func (r *RssRoot) OnAdd(ctx context.Context) {
	log.Printf("fetching RSS feed from: %s", r.feedUrl)
	r.items = make(map[string]*rss.Item)
	err := r.fetchRssData()
	if err != nil {
		log.Printf("error fetching RSS feed '%s': %v", r.feedName, err)
	} else {
		log.Printf("successfully fetched %d items from feed '%s'", len(r.items), r.feedName)
	}
}

func (r *RssRoot) fetchRssData() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	feed, err := rss.Fetch(r.feedUrl)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}
	if feed == nil {
		return fmt.Errorf("fetched feed is nil")
	}
	r.feedName = feed.Title

	if r.items == nil {
		r.items = make(map[string]*rss.Item)
	}

	for _, item := range feed.Items {
		if item == nil || item.Title == "" {
			log.Println("skipping item with missing title")
			continue
		}
		baseFilename := sanitizeFilename(item.Title)
		if baseFilename == "" {
			log.Printf("skipping item with sanitized title becoming empty: '%s'", item.Title)
			continue
		}
		if _, exists := r.items[baseFilename]; exists {
			log.Printf("warning: Duplicate sanitized base filename '%s', overwriting.", baseFilename)
		}
		r.items[baseFilename] = item
	}
	return nil
}

func (r *RssRoot) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries := make([]fuse.DirEntry, 0, len(r.items))
	for baseName := range r.items {
		entries = append(entries, fuse.DirEntry{
			Name: baseName + fileSuffix,
			Mode: fuse.S_IFREG,
		})
	}
	log.Printf("Readdir called, returning %d entries", len(entries))
	return fs.NewListDirStream(entries), fs.OK
}

func (r *RssRoot) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if !strings.HasSuffix(name, fileSuffix) {
		log.Printf("lookup: '%s' does not have expected suffix '%s'", name, fileSuffix)
		return nil, syscall.ENOENT
	}
	baseName := strings.TrimSuffix(name, fileSuffix)

	r.mu.Lock()
	item, ok := r.items[baseName]
	r.mu.Unlock()

	if !ok {
		log.Printf("lookup: base name '%s' (from '%s') not found in map", baseName, name)
		return nil, syscall.ENOENT
	}

	log.Printf("lookup: found base name '%s' (from '%s')", baseName, name)

	// Determine content: Prefer Content, fallback to Description/Summary
	var contentStr string
	if item.Content != "" {
		contentStr = item.Content
	} else if item.Content != "" {
		contentStr = item.Content
	} else if item.Summary != "" {
		contentStr = item.Summary
	} else {
		contentStr = "no content available."
	}
	contentBytes := []byte(fmt.Sprintf("Title: %s\nLink: %s\nPublished: %s\n\n%s\n", // in our case published will just be local time because the YSWS RSS feed doesn't have start dates I think
		item.Title,
		item.Link,
		item.Date.Format(time.RFC1123),
		contentStr,
	))

	fileNode := &RssFile{
		content: contentBytes,
	}

	stable := fs.StableAttr{Mode: fuse.S_IFREG}
	childInode := r.NewInode(ctx, fileNode, stable)

	out.Attr.Mode = 0444 
	out.Attr.Size = uint64(len(contentBytes))
	now := time.Now()
	attrTime := now
	if !item.Date.IsZero() {
		attrTime = item.Date
	}
	out.SetTimes(&attrTime, &now, &now)

	out.NodeId = childInode.StableAttr().Ino
	out.Generation = childInode.StableAttr().Gen

	return childInode, fs.OK
}

func (r *RssRoot) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0555 | fuse.S_IFDIR
	now := time.Now()
	out.SetTimes(&now, &now, &now)
	return fs.OK
}

func main() {
	defaultFeedURL := "https://ysws.hackclub.com/feed.xml" 

	debug := flag.Bool("debug", false, "enable FUSE debug logging")
	feedURL := flag.String("feed", defaultFeedURL, "URL of the RSS/Atom feed to fetch")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s <mountpoint>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	mountpoint := flag.Arg(0)

	root := &RssRoot{
		feedUrl: *feedURL,
	}

	opts := &fs.Options{
		MountOptions: fuse.MountOptions{
			Name:   "hackfs",
			Debug:  *debug,
			FsName: fmt.Sprintf("rss:%s", *feedURL),
		},
	}

	server, err := fs.Mount(mountpoint, root, opts)
	if err != nil {
		log.Fatalf("mount failed: %v", err)
	}

	log.Printf("filesystem mounted at %s.", mountpoint)
	log.Printf("press ctrl+c to unmount.")

	server.Wait()

	log.Printf("filesystem unmounted.")
}
