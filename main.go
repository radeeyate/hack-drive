package main

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall/js"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/xeonx/timeago"
)

var (
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#528bff"))

	// Styles for the welcome message.  Organized for readability.
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")). // Purple
			Padding(0, 1)                          // Padding around the text

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Italic(true).
			Padding(0, 4).
			Bold(true)

	bold = lipgloss.NewStyle().
		Bold(true)

	// Neofetch styles
	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF69B4")) // Hot Pink
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	startTime time.Time
)

var welcomeText = lipgloss.JoinVertical(
	lipgloss.Center,
	titleStyle.Render(
		`
  #                    #                 #           "                 
  # mm    mmm    mmm   #   m          mmm#   m mm  mmm    m   m   mmm  
  #"  #  "   #  #"  "  # m"          #" "#   #"  "   #    "m m"  #"  # 
  #   #  m"""#  #      #"#           #   #   #       #     #m#   #"""" 
  #   #  "mm"#  "#mm"  #  "m         "#m##   #     mm#mm    #    "#mm"
`),
	subtitleStyle.Render(
		"you ship us something cool using fuse, we'll ship you a hack club branded flash drive",
	),
	"\n",
	bold.Render(
		"type",
	)+lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CA402B")).
		Render(" help ")+
		bold.Render(
			"to get started",
		),
)

type model struct {
	textInput textinput.Model
	output    string
	err       error
	history   []string
	histIndex int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "enter a command; help to see commands"
	ti.Prompt = promptStyle.Render("$ ")
	ti.Focus()
	ti.Width = 20
	ti.Cursor.Style = cursorStyle

	return model{
		textInput: ti,
		output:    welcomeText + "\n\n",
		err:       nil,
		history:   []string{},
		histIndex: 0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			m.output += m.textInput.Prompt + m.textInput.Value() + "\n"

			if input != "" {
				if len(m.history) == 0 || m.history[len(m.history)-1] != input {
					m.history = append(m.history, input)
				}
			}
			m.histIndex = len(m.history)

			output := executeCommand(input)
			m.output += output + "\n"

			m.textInput.SetValue("")
			m.textInput.Focus()
			return m, nil

		case tea.KeyCtrlC:
			m.output += m.textInput.Prompt + m.textInput.Value() + "^C\n"
			m.textInput.SetValue("")
			return m, nil

		case tea.KeyCtrlL:
			m.output = ""
			return m, nil

		case tea.KeyTab:
			input := strings.TrimSpace(m.textInput.Value())

			if input != "" {
				commands := []string{"ls", "help", "cat", "cat about.txt", "cat resources.txt", "cat submit.txt", "cat requirements.txt", "uptime", "neofetch"} // Include neofetch in suggestions

				matches := fuzzy.Find(input, commands)
				if len(matches) > 0 {
					completed := matches[0]
					m.textInput.SetValue(completed)
					m.textInput.SetCursor(len(completed))
				}

				return m, nil
			}

		case tea.KeyUp:
			if m.histIndex > 0 {
				m.histIndex--
				m.textInput.SetValue(m.history[m.histIndex])
				m.textInput.SetCursor(len(m.history[m.histIndex]))
			}
			return m, nil

		case tea.KeyDown:
			if m.histIndex < len(m.history)-1 {
				m.histIndex++
				m.textInput.SetValue(m.history[m.histIndex])
				m.textInput.SetCursor(len(m.history[m.histIndex]))
			} else if m.histIndex == len(m.history)-1 {
				m.histIndex = len(m.history)
				m.textInput.SetValue("")
			}
			return m, nil

		}

	case tea.WindowSizeMsg:
		m.textInput.Width = msg.Width - 2
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf("%s%s", m.output, m.textInput.View())
}

func executeCommand(command string) string {
	switch command {
	case "ls":
		return "about.txt  resources.txt  requirements.txt  submit.txt"
	case "cat about.txt":
		return `hack drive is the ysws about making a cool filesystem. have
you heard of fuse (filesystem in userspace)? it's a way to
make a filesystem without needing to write kernel code. you
write a program with either fuse, winfsp, fuse-t, macfuse, or
another tool of your choosing. maybe try implementing a
harder drive (http://tom7.org/harder/), or make a slack
client (please please please!) where you send and read 
messages by making files. it's up to you, there's endless 
possibilities. make a fedi client? store your files by 
encoding and uploading them to youtube first? or maybe you
want to store your files to a blockchain? it's all up to
you, we encourage creativity.

you're probably wondering, though, "what do i get in
return?"

good question!

as long as you spend 3-4 hours workinng on your project
we'll send you a custom hack club branded flash drive. if
you make something really cool that we like, maybe an ssd!
make sure to log your time on hackatime, or else your time
spent will be invalid.

once you create your project, please see the submission
instructions in submit.txt.
`
	case "cat resources.txt":
		return `here are some resources to help you get started:
  - fuse (c): https://github.com/libfuse/libfuse
    - awesome-fuse-fs: https://github.com/koding/awesome-fuse-fs
    - go-fuse (golang): https://github.com/hanwen/go-fuse
    - pyfuse3 (python3): https://github.com/libfuse/pyfuse3
    - fuser (rust): https://github.com/cberner/fuser
	- jnr-fuse (java): https://github.com/SerCeMan/jnr-fuse
  - winfsp: https://github.com/billziss-gh/winfsp
    - cgofuse (golang; also works for fuse and macfuse): https://github.com/winfsp/cgofuse
	- winfsp-rs (rust): https://github.com/SnowflakePowered/winfsp-rs
	- jnr-winfsp (java): https://github.com/jnr-winfsp-team/jnr-winfsp
	- winfspy (python): https://github.com/Scille/winfspy
  - fuse-t: https://github.com/macos-fuse-t/fuse-t
    - compatible with other fuse libraries (theoretically)
  - harder drive: http://tom7.org/harder/
  - fuse wikipedia entry: https://en.wikipedia.org/wiki/Filesystem_in_Userspace
  - hackatime: https://waka.hackclub.com/
`
	case "cat submit.txt":
		return `
to submit your project, please do the following:

1. make sure you've logged at least 3-4 hours on hackatime.
2. make a github repo with your project.
3. make a demo and submit your project on this form: https://google.com
5. wait for us to review your submission.
6. if your submission is accepted, we'll send you a flash drive!
`
	case "cat requirements.txt":
		return `the following requirements must be met to receive a flash drive:
  - spend 3-4 hours working on your project
  - your project must be open source
  - your project must be mountable and functional
  - your project must support at least the following operations:
    - getattr
	- create
	- open
	- read
	- write
	- readdir
  - data cannot be stored conventionally you cannot:
    - store data on a local disk or in a RAM disk
  - acceptable and encouraged ideas:
    - storing files on youtube by encoding them into video frames
	- slack client interacted with via the filesystem
	- store files on a blockchain
	- encoding files into qr codes
	- a fediverse client
	- or any other weird thing you can think of
  - your filesystem doesn't have to store files in the traditional sense.
  for example, you could create a filesystem that acts as a slack client,
  where creating a file sends a message, and reading a file displays messages.
  the key is to present a filesystem interface, even if the underlying data
  isn't typical "files"
  `
	case "help":
		return "Commands: help, ls, cat, neofetch"
	case "neofetch":
		return neofetch()
	case "uptime": // TODO: After launch (hopefully!) this should instead be a countdown for the YSWS end date
		return strings.Replace(timeago.English.Format(startTime), " ago", "", 1)
	}

	return ""
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func neofetch() string {
	// Get system information.  More could be added, but this is a good start.
	os := runtime.GOOS
	arch := runtime.GOARCH
	goVersion := runtime.Version()
	numCPU := runtime.NumCPU()

	// Create the ASCII art.  This is a simplified Hack Club logo.  Could be made more complex.
	asciiArt :=
		`@@G5J??????JJJY5P#@@
&J77??7!!?JJJJJJJYP&
Y7????^ .?JJJJJYYYYP
?????J^ .~^^~?YYYYYY
?JJJJJ^  !7. ^YYYYYY
JJJJJJ^ .JJ. ^YYYYY5
5JJJJY^ :JJ: ^YYYYYP
&5YYYYJ?JYYJ?JYYYYP&
@@#GP555YYYY555PG#@@
`

	artStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f1494a"))

	// Build the output string using lipgloss styles for formatting.
	output := lipgloss.JoinHorizontal(
		lipgloss.Top, // Align the text and ASCII art at the top.
		artStyle.Render(asciiArt),
		"   ", // Add some spacing between the info and the art.
		lipgloss.JoinVertical(
			lipgloss.Left, // Left-align the system info.
			labelStyle.Render("localhost")+"@"+labelStyle.Render("hack-drive"),
			"--------------------",
			labelStyle.Render("OS:      ")+infoStyle.Render(os),
			labelStyle.Render("Arch:    ")+infoStyle.Render(arch),
			labelStyle.Render("Go:      ")+infoStyle.Render(goVersion),
			labelStyle.Render("CPU:     ")+infoStyle.Render(fmt.Sprintf("%d", numCPU)),
			labelStyle.Render(
				"Uptime:  ",
			)+infoStyle.Render(
				strings.Replace(timeago.English.Format(startTime), " ago", "", 1),
			),
		)+"\n\n",
	)

	return output
}

type MinReadBuffer struct {
	buf *bytes.Buffer
}

// for some reason bubbletea doesn't like a Reader that will return 0 bytes instead of blocking,
// so we use this hacky workaround for now.
func (b *MinReadBuffer) Read(p []byte) (n int, err error) {
	for b.buf.Len() == 0 {
		time.Sleep(100 * time.Millisecond)
	}
	return b.buf.Read(p)
}

func (b *MinReadBuffer) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func createTeaForJS(model tea.Model, option ...tea.ProgramOption) *tea.Program {
	fromJs := &MinReadBuffer{buf: bytes.NewBuffer(nil)}
	fromGo := bytes.NewBuffer(nil)

	prog := tea.NewProgram(
		model,
		append([]tea.ProgramOption{tea.WithInput(fromJs), tea.WithOutput(fromGo)}, option...)...)

	// Register write function in WASM
	js.Global().Set("bubbletea_write", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fromJs.Write([]byte(args[0].String()))
		fmt.Println("Wrote to Go:", args[0].String())
		return nil
	}))

	// Register read function in WASM
	js.Global().Set("bubbletea_read", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		b := make([]byte, fromGo.Len())
		_, _ = fromGo.Read(b)
		fromGo.Reset()
		if len(b) > 0 {
			fmt.Println("Read from Go:", string(b))
		}
		return string(b)
	}))

	// Register resize function in WASM
	js.Global().Set("bubbletea_resize", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		width := args[0].Int()
		height := args[1].Int()
		prog.Send(tea.WindowSizeMsg{Width: width, Height: height})
		return nil
	}))

	return prog
}

func main() {
	startTime = time.Now()

	prog := createTeaForJS(initialModel(), tea.WithAltScreen())

	fmt.Println("Starting program...")
	if _, err := prog.Run(); err != nil {
		fmt.Println("Error while running program:", err)
		os.Exit(1)
	}
}
