# qedit

qedit is a simple and easy to maintain editor, designed to run portably on as many
operating systems and architectures as possible. It does so by leveraging the cross-compile
features of Go, and the very portable [tcell](https://github.com/gdamore/tcell)
terminal library. A side effect of using Go is the absolute simplicity with which
the project can be maintained. For an open source project under the very permissive
MIT license, this is important.

## Table of Contents

 - [Goals](#goals)
 - [Screenshots](#screenshots)
 - [Building](#building)
 - [Contributing](#contributing)
 - [FAQ](#for-answers-and-questions-%28FAQ%29)

---

## Goals

 * DOS-like user interface library for Go/tcell (diesel)
 * Modern [rope](https://en.wikipedia.org/wiki/Rope_(data_structure)) buffer (used in emacs)
 * Modern text editing, including: copy/paste, mouse support, selection, etc.
 * Btree-based tiling and floating window management (panels)
 * Extensions (via IPC or the planned [scripting language](https://github.com/d5/tengo))
 * Built-in terminal

## Screenshots

![Editing the README with the "Panel" menu open.](/screenshots/qedit-alpha-dev-panel-menu.png)
![Showing the "Open files" dialog.](/screenshots/qedit-alpha-dev-open-files-dialog.png)
![Showing the "Edit" menu with selected text.](/screenshots/qedit-alpha-dev-copy-selection.png)

## Building

You will need:

 * A clone or download of this repository
 * A Go compiler version supporting Go 1.15+
 * A temporary internet connection

With Go successfully in your path variable, open a terminal and navigate to the
directory containing `main.go`, and execute the following command:

```
go build
```

That will download and install dependencies, and build the binary named `qedit`
or `qedit.exe`. If you would like Go to install and manage the binary for you,
which will add it to GOPATH/bin (and therefore your path variable):

```
go install
```

You can now run `qedit` anywhere.

## Contributing

I love contributions!

Currently, I am focused on working solo, so there aren't many issues authored and I'm
doing most work quietly. If I know there are people interested in contributing, I will
change how I am working on qedit.

As for types of contributions: I am currently seeking bug fixes, and resolving all of
my `TODO` comments in the code. Or possibly removing them and turning them into issues.
Possibly the best contribution I could receive is someone using my editor as a daily-driver
(bless them for their troubles) and reporting issues or improvements.

If you would like to communicate, you can message me on Matrix: `@fivemoreminix:matrix.org`,
or Discord: `fivemoreminix#7637`, or even email: thelukaswils ATT (the google one)

## For Answers and Questions (FAQ)

### Why another text editor?
I like using my own software. I also like retro aesthetic, so I designed my own
modern text editor around the simple and stylish MS-DOS EDIT.COM editor. I am
pleased to use it for all of my text editing purposes, and it is what 80% of qedit
has been edited with.

I made qedit simple and easy to maintain, and MIT licensed, so anyone like
you can make a fork and have your own personal editor.

### Why Go?
Truthfully, I spent about a year delaying work on the editor, because I didn't want
to use Go, Python, or any other language. I wanted to use Zig, because it was portable,
fast, and efficient. I spent months working on [termelot](https://github.com/minierolls/termelot),
a terminal library similar to tcell, but for Zig. When Zig is ready, I will continue
work on the library, but this editor is likely to forever remain in Go.

I finally came around when I decided that the tcell library is excellent, and would
make programming my editor easy. I started working, and I haven't regretted choosing Go,
and the performance of the editor is fantastic.
