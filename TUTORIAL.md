# Tutorial

So, I've heard you wanted to make a custom filesystem using FUSE!

I will be using Golang in this tutorial, but you can use any language you want. If you don't want to use Go, there is a variety of resources listed on the [website](https://hackdrive.radi8.dev).

## Setting up your environment

There are many different ways to install FUSE or similar libraries across systems. You can find a guide by searching up something like `"install fuse ubuntu"` or `"how to install macfuse"`.

### Go

This tutorial assumes you have Golang already installed. If you don't, you can find instructions here: <https://go.dev/doc/install>.

### Setting up your project

```bash
mkdir hackfuse
cd hackfuse
go mod init hackfuse
go get github.com/hanwen/go-fuse/v2/fs
go get github.com/axgle/mahonia
go get github.com/SlyMarbo/rss
```

Feel free to change the project name to your choosing.

I want to make a filesystem that pulls the latest Hack Club YSWS Programs and tells me everything I need to know about them. I'll be using [rss](https://github.com/SlyMarbo/rss) to fetch and parse them. After that, the filesystem will create fake files to be served to the user.

### The fun stuff :)

Every `.go` file must start with a package declaration. Usually, this will be `main`, but can change based on project requirements

I'll also add all my imports here.

```go
import (
   "context" // for fuse operations context
   "flag"    // command line argument parsing
   "fmt"     // formatting i/o
   "log"     // logging
   "os"      // interactions with the os (e.g exit)
   "strings" // string manipulation
   "sync"    // for mutexes (protecting shared data)
   "syscall" // fuse error codes (errno)
   "time"    // timestamping
  
   // fuse libraries
   "github.com/hanwen/go-fuse/v2/fs"
   "github.com/hanwen/go-fuse/v2/fuse"
  
   // rss parsing library
   "github.com/SlyMarbo/rss"
)
```

### Constants and Helper Functions

We need a function to sanitize and add `.txt` to the end of YSWS names.

```go
const fileSuffix = ".txt"

func sanitizeFilename(name string) string {
    name = strings.ReplaceAll(name, "/", "-")
    name = strings.TrimSpace(name)
    return name
}
```

* `fileSuffix` is the extension for the ysws names
* `sanitizeFilename` cleans up titles

### `RssFile` Node

`RssFile` represents a single file (or in this case, a YSWS) in our filesystem. It must handle operatings including opening, reading, and getting attributes.

```go
type RssFile struct {
        fs.Inode
        content []byte
}

// we must implement the following interfaces to use FUSE
var _ = (fs.NodeOpener)((*RssFile)(nil))
var _ = (fs.NodeGetattrer)((*RssFile)(nil))
var _ = (fs.FileReader)((*RssFile)(nil))

func (f *RssFile) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
    end := int(off) + len(dest)
    if end > len(f.content) {
        end = len(f.content)
    }

    // check for reading past EOF
    if off >= int64(len(f.content)) {
        return fuse.ReadResultData([]byte{}), fs.OK
    }

    // check for invalid offset
    if off < 0 {
        return nil, syscall.EINVAL
    }    

    // return the requested slice
    return fuse.ReadResultData(f.content[off:end]), fs.OK

}

// open is called when the file is opened
// for just a read-only file, we return the node itself as a file handle.
func (f *RssFile) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
        return f, 0, fs.OK
}

// getattr gets file attributes (permissions, size, timestamps, etc.)
func (f *RssFile) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
        out.Mode = 0444 // read only permissions (-r--r--r--)
        out.Size = uint64(len(f.content)) // the file size is the length of our content buffer
        now := time.Now()
        out.SetTimes(&now, &now, &now)
        return fs.OK // return success
}
```

* We embed `fs.Inode` which is standard practice in `go-fuse`
* `content` stores the YSWS data
* The `var _ = ...` lines are a compile time check to make sure we did things properly
* `Read` adds the logic to return chunks of `content`
* `Getattr` gives metadata to files

### `RssRoot` Node

This structs represents the root directory of our filesystem. It fetches the RSS feed and handles directory listings (`Readdir`) and looking up specific files (`Lookup`).

```go
type RssRoot struct {
    fs.Inode
     
    mu       sync.Mutex        // mutex to protect access to items map
    items    map[string]*rss.Item // map from sanitized filename base -> RSS item
    feedUrl  string
    feedName string
}

var _ = (fs.NodeOnAdder)((*RssRoot)(nil))   // called when node is added to tree
var _ = (fs.NodeReaddirer)((*RssRoot)(nil)) // can list directory contents
var _ = (fs.NodeLookuper)((*RssRoot)(nil))  // can look up children by name
var _ = (fs.NodeGetattrer)((*RssRoot)(nil)) // can get directory attributes

// onadd is called when the filesystem is mounted and the root node is added
// it is usually a good place to make initial setup - in our case, fetching the feed
func (r *RssRoot) OnAdd(ctx context.Context) {
    log.Printf("fetching RSS feed from: %s", r.feedUrl)
    r.items = make(map[string]*rss.Item) // initialize the map
    err := r.fetchRssData()
    if err != nil {
        log.Printf("error fetching RSS feed '%s': %v", r.feedUrl, err)
    } else {
    log.Printf("successfully fetched %d items from feed '%s'", len(r.items), r.feedName)
    }
}

// handles fetching and parsing the YSWS list
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

    // make sure items map is initialized
    if r.items == nil {
        r.items = make(map[string]*rss.Item)
    }

    // clear old items before adding new ones
    for k := range r.items {
        delete(r.items, k)
    }

    // iterate through fetched programs
    for _, item := range feed.Items {
        if item == nil || item.Title == "" {
            log.Println("skipping item with missing title")
            continue
        }
        // Sanitize the title to create a base filename
        baseFilename := sanitizeFilename(item.Title)
        if baseFilename == "" {
            log.Printf("skipping item whose sanitized title is empty: original='%s'", item.Title)
            continue // skip if sanitization results in an empty string
        }

        // check for potential filename collisions after sanitization
        if _, exists := r.items[baseFilename]; exists {
            log.Printf("warning: duplicate sanitized base filename '%s' from title '%s'; overwriting.", baseFilename, item.Title)
        }

        r.items[baseFilename] = item
    }

    return nil // success
}

// readdir lists the contents of the directory - in this case, all programs
func (r *RssRoot) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
        r.mu.Lock()
        defer r.mu.Unlock()

        entries := make([]fuse.DirEntry, 0, len(r.items))
        for baseName := range r.items {
                entries = append(entries, fuse.DirEntry{
                        Name: baseName + fileSuffix,
                        Mode: fuse.S_IFREG, // indicate it's a regular file
                })
        }
        log.Printf("Readdir called, returning %d entries", len(entries))
        return fs.NewListDirStream(entries), fs.OK
}

// lookup finds a specific child node (a file in this case) by name.
func (r *RssRoot) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
        // does the requested name have our expected suffix?
        if !strings.HasSuffix(name, fileSuffix) {
                log.Printf("lookup: '%s' does not have expected suffix '%s'", name, fileSuffix)
                return nil, syscall.ENOENT // "no such file or directory"
        }
        baseName := strings.TrimSuffix(name, fileSuffix)

        r.mu.Lock()
        item, ok := r.items[baseName]
        r.mu.Unlock()

        if !ok {
                log.Printf("lookup: Base name '%s' (from '%s') not found in map", baseName, name)
                return nil, syscall.ENOENT // "no such file or directory"
        }

        log.Printf("Lookup: Found base name '%s' (from '%s')", baseName, name)


        var contentStr string
        if item.Content != "" {
                contentStr = item.Content
        } else if item.Summary != "" {
                contentStr = item.Summary
        } else {
                contentStr = "No content available."
        }

        contentBytes := []byte(fmt.Sprintf("Title: %s\nLink: %s\nPublished: %s\n\n%s\n",
                item.Title,
                item.Link,
                item.Date.Format(time.RFC1123),
                contentStr,
        ))

        // create the file node instance, and setting the content
        fileNode := &RssFile{
                content: contentBytes,
        }

        // define stable attributes for the new inode (just the mode here)
        // stable attributes don't change often. S_IFREG means a regular file
        stable := fs.StableAttr{Mode: fuse.S_IFREG}

        // create a new Inode associated with our RssFile node.
        // this links the filesystem object (inode) to our custom logic (fileNode)
        // the inode is created as a child of the current node (r)
        childInode := r.NewInode(ctx, fileNode, stable)

        // these attributes are returned immediately to the kernel for the lookup operation.
        out.Attr.Mode = 0444 // read-only file mode
        out.Attr.Size = uint64(len(contentBytes))
        now := time.Now()
        attrTime := now
        if !item.Date.IsZero() {
                attrTime = item.Date
        }

        out.SetTimes(&attrTime, &attrTime, &now)

        // set timeouts for how long the kernel should cache this entry information
        // default timeouts are often sufficient

        // link the output EntryOut to the newly created Inode ID and Generation
        // this tells the kernel which inode corresponds to the name looked up
        out.NodeId = childInode.StableAttr().Ino
        out.Generation = childInode.StableAttr().Gen

        return childInode, fs.OK // success!
}
// getattr for the root directory itself.
func (r *RssRoot) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
    out.Mode = 0555 | fuse.S_IFDIR // read/execute permissions for a directory
    now := time.Now()
    out.SetTimes(&now, &now, &now)
    return fs.OK // success!
}
```

* `OnAdd` fetches initial RSS data after mounted
* `Readdir` interates of the YSWS list and returns a list
* `Lookup`:
  * checks if the YSWS is in the `items` map
  * if found, formats the content
  * creates an `RssFile` node instance with that content
  * populates `fuse.EntryOut` with attributes
  * returns the new inode
* `Getattr` returns the attributes if the root directory

### The Entry Point
```go
func main() {
    // default feed url if none is provided via flags
    defaultFeedURL := "https://ysws.hackclub.com/feed.xml"


    debug := flag.Bool("debug", false, "enable FUSE debug logging")
    feedURL := flag.String("feed", defaultFeedURL, "URL of the RSS/Atom feed to fetch")
    flag.Parse()

    // check if the mountpoint argument is provided
    if flag.NArg() != 1 {
        fmt.Fprintf(os.Stderr, "usage: %s [options] <mountpoint>\n", os.Args[0])
        flag.PrintDefaults()
        os.Exit(1)
    }
    mountpoint := flag.Arg(0)

    // create the root node instance
    root := &RssRoot{
        feedUrl: *feedURL,
    }

    // configure FUSE mount options
    opts := &fs.Options{
        MountOptions: fuse.MountOptions{
            Name:   "hackfs",
            Debug:  *debug, // enable debug logging if requested
            FsName: fmt.Sprintf("rss:%s", *feedURL),
        },
    }

    // mount the filesystem
    // fs.Mount takes the mountpoint, the root node, and options
    server, err := fs.Mount(mountpoint, root, opts)
    if err != nil {
        log.Fatalf("mount failed: %v", err)
    }

    log.Printf("filesystem mounted at %s.", mountpoint)
    log.Printf("using feed: %s", *feedURL)
    log.Printf("press ctrl+c to unmount.")

    // wait for the filesystem server to finish (e.g. on unmount)
    server.Wait()

    log.Printf("Filesystem unmounted.")
}
```
