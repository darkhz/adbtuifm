[![Go Report Card](https://goreportcard.com/badge/github.com/darkhz/adbtuifm)](https://goreportcard.com/report/github.com/darkhz/adbtuifm)
# adbtuifm

![demo](demo/demo.gif)

adbtuifm is a TUI file manager for ADB.

It supports:
- Copy, move, and delete operations on the ADB device and the local machine
- Transferring files/folders between the ADB device and the local machine

It has been tested only on Linux. Windows/Mac is currently not supported.

Note that this is an experimental release, so expect bugs.

# Installation
```
go get -u github.com/darkhz/adbtuifm
```
# Usage
```
adbtuifm [<flags>]

Flags:
  --mode <Local/ADB>  Specify which mode to start in
  --remote=<path>     Specify the remote(ADB) path to start in
  --local=<path>      Specify the local path to start in
  ```

# Keybindings
**Note: Only Copy operations are cancellable, Move and Delete operations will persist**

## Main Page
|Operation                              |Key                          |
|---------------------------------------|-----------------------------|
|Switch between panes                   |<kbd>Tab</kbd>               |
|Navigate between entries               |<kbd>Up</kbd>/<kbd>Down</kbd>|
|Switch between ADB/Local (in each pane)|<kbd>s</kbd>                 |
|Switch to operations page              |<kbd>o</kbd>                 |
|Select directory/file                  |<kbd>Enter</kbd>             |
|Change one directory back              |<kbd>Backspace</kbd>         |
|Change to any directory                |<kbd>g</kbd>                 |
|Refresh                                |<kbd>r</kbd>                 |
|Copy                                   |<kbd>c</kbd>                 |
|Move                                   |<kbd>m</kbd>                 |
|Paste/Put                              |<kbd>p</kbd>                 |
|Delete                                 |<kbd>d</kbd>                 |
|Cancel Copy/Move                       |<kbd>Esc</kbd>               |
|Toggle hidden files                    |<kbd>h</kbd>                 |
|Quit                                   |<kbd>q</kbd>                 |

## Operations Page
|Operation                |Key                          |
|-------------------------|-----------------------------|
|Navigate between entries |<kbd>Up</kbd>/<kbd>Down</kbd>|
|Switch to main page      |<kbd>o</kbd>/<kbd>Esc</kbd>  |
|Cancel selected operation|<kbd>x</kbd>                 |
|Cancel all operations    |<kbd>X</kbd>                 |

# todo
- Remove the file after an operation has been cancelled
- Make error messages more informative
- Refactor the code, eliminate code duplication
- Restructure the UI
