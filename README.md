Diesel is an attractive and powerful text editor for the command-line, that runs on any operating system. The Diesel editor is a single, small binary without any more features than file handling, mouse support, basic text editing, and an extensive set of user interface tools. But what really makes Diesel so interesting is the package manager that comes included. Accessible from the toolbar, it allows you to download and compile extensions written in Zig, C, C++, or any other variety of languages (with C FFI). Extensions can add features like:

 - Syntax highlighting
 - Integrated terminals
 - Autocompletion
 - Hex / machine code viewing and editing
 - Build tasks
 - VIM / Emacs / etc. keymaps and systems
 - Breakout / Tetris / Space Invaders / Snake
 - Debugging
 - Git clients
 - you name it

And it's easy to add your own commands and shortcuts to your toolbar.

> It is up to extensions to implement the features users want to see. And every extension is unique -- each its own build system; each its own ideal optimization: of speed, size, or safety; each its own codebase. But all extensions are compiled right on your machine, in the Gentooman-style, to take advantage of all your CPU's glorious features. Diesel will be the fastest text editor you use!

Go to the [Releases](https://github.com/codemessiah/diesel/releases) page to download the latest version of Diesel for free and start using it today!

# Package Manager and Extensions
The package manager is fundamental to the design of Diesel -- it provides opt-in features, to save yourself from the everyday strike of bloatware that has infected the software industry since its inception. (See Wirth's Law) Diesel is optimized for size and stability. It offers a base set of shared features which extensions freely rely upon. It is up to extensions to implement the features users want to see. And every extension is unique -- each its own build system; each its own ideal optimization: of speed, size, or safety; each its own codebase. But all extensions are compiled right on your machine, in the Gentooman-style, to take advantage of all your CPU's glorious features. Diesel will be the fastest text editor you use!

## Making and Publishing Extensions
Diesel is just as much your editor as it is mine. If you know some Zig, C, or C++ programming you can get started making extensions for Diesel right away! Although making extensions with other languages, like Python, Lua, Go, or C# is trickier, it is NOT impossible! With a little C FFI magic, any language can be used to make extensions with diesel.

See the Developing Extensions page on the Wiki for more informatioN!
 
