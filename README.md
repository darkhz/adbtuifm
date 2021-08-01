
[![Go Report Card](https://goreportcard.com/badge/github.com/darkhz/adbtuifm)](https://goreportcard.com/report/github.com/darkhz/adbtuifm)
# adbtuifm

![demo](demo/demo.gif)

adbtuifm is a TUI-based file manager for the Android Debug Bridge (ADB),
to make transfers between the device and client easier.

It has been tested only on Linux. Windows/Mac is currently not supported.

Note that this is an experimental release, so expect bugs.

# Features
- Transferring files/folders between the ADB device and the local machine
- Copy, move, and delete operations on the ADB device and the local machine seperately
- Ability to change to any directory via an inputbox, with autocompletion support
- Ability to select multiple items and perform various operations on them (multiselection mode)

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
|Operation                                 |Key                          |
|------------------------------------------|-----------------------------|
|Switch between panes                      |<kbd>Tab</kbd>               |
|Navigate between entries                  |<kbd>Up</kbd>/<kbd>Down</kbd>|
|Change directory to highlighted entry     |<kbd>Enter</kbd>             |
|Change one directory back                 |<kbd>Backspace</kbd>         |
|Switch between ADB/Local (in each pane)   |<kbd>s</kbd>                 |
|Switch to operations page                 |<kbd>o</kbd>                 |
|Change to any directory                   |<kbd>g</kbd>                 |
|Refresh                                   |<kbd>r</kbd>                 |
|Copy                                      |<kbd>c</kbd>                 |
|Move                                      |<kbd>m</kbd>                 |
|Paste/Put                                 |<kbd>p</kbd>                 |
|Delete                                    |<kbd>d</kbd>                 |
|Toggle hidden files                       |<kbd>h</kbd>                 |
|Multiselect mode (select one item)        |<kbd>S</kbd>                 |
|Multiselect mode (select all items)       |<kbd>A</kbd>                 |
|Cancel pending operation/ Reset selections|<kbd>Esc</kbd>               |
|Quit                                      |<kbd>q</kbd>                 |

## Operations Page
|Operation                |Key                          |
|-------------------------|-----------------------------|
|Navigate between entries |<kbd>Up</kbd>/<kbd>Down</kbd>|
|Switch to main page      |<kbd>o</kbd>/<kbd>Esc</kbd>  |
|Cancel selected operation|<kbd>x</kbd>                 |
|Cancel all operations    |<kbd>X</kbd>                 |
|Clear operations list    |<kbd>C</kbd>                 |

## Autocompletion Inputbox
|Operation                            |Key                              |
|-------------------------------------|---------------------------------|
|Navigate between entries             |<kbd>Up</kbd>/<kbd>Down</kbd>    |
|Autocomplete                         |<kbd>Tab</kbd>/<kbd>Any key</kbd>|
|Change directory to highlighted entry|<kbd>Enter</kbd>                 |
|Move back a directory                |<kbd>Ctrl</kbd>+<kbd>W</kbd>     |
|Switch to main page                  |<kbd>Esc</kbd>                   |

## Confirm/Error Dialog
|Operation                          |Key                             |
|-----------------------------------|--------------------------------|
|Switch between textview and buttons|<kbd>Left</kbd>/<kbd>Right</kbd>|
|Scroll in textview                 |<kbd>Up</kbd>/<kbd>Down</kbd>   |
|Select highlighted button          |<kbd>Enter</kbd>                |

# Bugs
-  In directories with a huge amount of entries, autocompletion will lag.
   This happens only on the device side (i.e ADB mode), where there is
   significant latency in transferring and processing the directory listing
   to the client.

