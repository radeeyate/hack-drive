# Hack Drive

Hack Drive is a proposed YSWS for Hack Club members where you create a custom filesystem using FUSE (Filesystem in Userspace) or a similar technology (WinFSP, fuse-t, MacFUSE). One thing - Your filesystem shouldn't use traditional storage methods!

The goal is to build a functional, mountable filesystem that implements a set of core operations, but uses a *non-conventional* data storage or retrieval mechanism.

Inspired by the "harder drives" concept by Tom Murphy VII (<http://tom7.org/harder/>).


## You Ship

You'll be creating a program that implements a filesystem.  This means users should be able to:

1. **Mount** your filesystem to a directory on their computer
2. **Interact** with it like a normal filesystem (create files, read files, list directories, etc.)
3. **Store (or retrieve)** data in a unconventional way

## We Ship

* **Everyone who meets the requirements:** A custom Hack Club branded flash drive!
* **Top Submission (chosen by a judging panel):** A 1TB Kingston XS2000 External SSD!

## Requirements

* Spend at least 3-4 hours working on your project.  Log your time using Hackatime (<https://waka.hackclub.com/>).
* Your project *must* be open source (e.g., on GitHub).
* he project must implement a *mountable and functional* filesystem.
* Your filesystem *must* support at least the following operations:
  * `getattr` (get file attributes)
  * `create` (create a new file)
  * `open` (open a file)
  * `read` (read from a file)
  * `write` (write to a file)
  * `readdir` (list files in a directory)
* The data storage method *cannot* be conventional. You cannot:
  * Store data on a local disk or in a RAM disk
* Acceptable and encouraged ideas:
  * Storing files on YouTube (by encoding them into video frames)
  * A Slack client interacted with via the filesystem
  * Storing files on a blockchain
  * Encoding files into QR codes
  * A Fediverse client
  * *Any other weird thing you can think of*
* Your filesystem doesn't *have* to store files in the traditional sense. For example, you could create a filesystem that acts as a Slack client, where creating a file sends a message, and reading a file displays messages.  The key is to present a filesystem *interface* to the user, even if the underlying data isn't typical "files."

## Tools and Libraries

You can use any of the following (or a similar tool):

* **FUSE (Linux):**  A widely used framework for creating filesystems in userspace
  * C:  [libfuse](https://github.com/libfuse/libfuse)
  * Go:  [go-fuse](https://github.com/hanwen/go-fuse)
  * Python:  [pyfuse3](https://github.com/libfuse/pyfuse3)
  * Rust:  [fuser](https://github.com/cberner/fuser)
  * Java: [jnr-fuse](https://github.com/SerCeMan/jnr-fuse)
  * [awesome-fuse-fs](https://github.com/koding/awesome-fuse-fs) - A list of FUSE filesystems and tools
* **WinFSP (Windows):**  The Windows equivalent of FUSE
  * [WinFSP](https://github.com/billziss-gh/winfsp)
  * Go (also works with FUSE and MacFUSE):  [cgofuse](https://github.com/winfsp/cgofuse)
  * Rust: [winfsp-rs](https://github.com/SnowflakePowered/winfsp-rs)
  * Java: [jnr-winfsp](https://github.com/jnr-winfsp-team/jnr-winfsp)
  * Python: [winfspy](https://github.com/Scille/winfspy)
* **fuse-t (macOS):**  A newer FUSE implementation for macOS.  (Should be compatible with many existing FUSE libraries)
  * [fuse-t](https://github.com/macos-fuse-t/fuse-t)
* **MacFUSE (macOS):**  The older FUSE implementation for macOS

## Resources

* [FUSE Wikipedia Entry](https://en.wikipedia.org/wiki/Filesystem_in_Userspace) - A good overview of FUSE.
* [Harder Drive](http://tom7.org/harder/) - The inspiration for this YSWS
* [Hackatime](https://hackatime.hackclub.com/) - Time tracking for Hack Clubbers
* The "resources.txt" file within the YSWS website provides links to many helpful libraries

## Contributing

The YSWS's website is written in Golang using the bubbletea TUI framework, and is built for WASM to run in browsers. To start development, clone the repository (or your fork), install dependencies, and run `air`:

```sh
$ git clone https://github.com/radeeyate/hack-drive.git
$ cd hack-drive
$ go mod tidy
$ air
```

If you'd like to contribute, please make a pull request! Any help is always appreciated.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
