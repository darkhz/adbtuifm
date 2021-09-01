

[![Go Report Card](https://goreportcard.com/badge/github.com/darkhz/adbtuifm)](https://goreportcard.com/report/github.com/darkhz/adbtuifm)
# adbtuifm

![demo](demo/demo.gif)

adbtuifm is a TUI-based file manager for the Android Debug Bridge, to make transfers
between the device and client easier.

It has been tested only on Linux. Windows/Mac is currently not supported.

# Features
- Multiselection support, similar to nnn

- Transferring files/folders between the device and the local machine

- Copy, move, and delete operations on the device and the local machine
  seperately

- Filter entries in each directory

- Rename files/folders or create directories

- Change to any directory via an inputbox, with autocompletion support

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
**Note:**<br />
**- As of v0.3.0, keybindings have been revised.**<br />
**- More information about an entry will be shown only in the top-down layout**<br />
**- Only Copy operations are cancellable, Move and Delete operations will persist**

## Main Page
|Operation                                 |Key                                 |
|------------------------------------------|------------------------------------|
|Switch between panes                      |<kbd>Tab</kbd>                      |
|Navigate between entries                  |<kbd>Up</kbd>/<kbd>Down</kbd>       |
|Change directory to highlighted entry     |<kbd>Enter</kbd>/<kbd>Right</kbd>   |
|Change one directory back                 |<kbd>Backspace</kbd>/<kbd>Left</kbd>|
|Switch to operations page                 |<kbd>o</kbd>                        |
|Switch between ADB/Local (in each pane)   |<kbd>s</kbd>/<kbd><</kbd>           |
|Change to any directory                   |<kbd>g</kbd>/<kbd>></kbd>           |
|Toggle hidden files                       |<kbd>h</kbd>/<kbd>.</kbd>           |
|Refresh                                   |<kbd>r</kbd>                        |
|Move                                      |<kbd>m</kbd>                        |
|Put/Paste (duplicate existing entry)      |<kbd>p</kbd>                        |
|Put/Paste (don't duplicate existing entry)|<kbd>P</kbd>                        |
|Delete                                    |<kbd>d</kbd>                        |
|Filter entries                            |<kbd>/</kbd>                        |
|Clear filtered entries                    |<kbd>Ctrl+r</kbd>                   |
|Select one item                           |<kbd>,</kbd>                        |
|Inverse selection                         |<kbd>a</kbd>                        |
|Select all items                          |<kbd>A</kbd>                        |
|Edit selection list                       |<kbd>S</kbd>                        |
|Make directory                            |<kbd>M</kbd>                        |
|Toggle top-down/right-left layout         |<kbd>[</kbd>                        |
|Swap panes                                |<kbd>]</kbd>                        |
|Rename files/folders                      |<kbd>R</kbd>                        |
|Reset selections                          |<kbd>Esc</kbd>                      |
|Help                                      |<kbd>?</kbd>                        |
|Quit                                      |<kbd>q</kbd>                        |
## Operations Page
|Operation                |Key                          |
|-------------------------|-----------------------------|
|Navigate between entries |<kbd>Up</kbd>/<kbd>Down</kbd>|
|Cancel selected operation|<kbd>x</kbd>                 |
|Cancel all operations    |<kbd>X</kbd>                 |
|Clear operations list    |<kbd>C</kbd>                 |
|Switch to main page      |<kbd>o</kbd>/<kbd>Esc</kbd>  |

## Change Directory InputBox
|Operation                            |Key                              |
|-------------------------------------|---------------------------------|
|Navigate between entries             |<kbd>Up</kbd>/<kbd>Down</kbd>    |
|Autocomplete                         |<kbd>Tab</kbd>/<kbd>Any key</kbd>|
|Change directory to highlighted entry|<kbd>Enter</kbd>                 |
|Move back a directory                |<kbd>Ctrl</kbd>+<kbd>W</kbd>     |
|Switch to main page                  |<kbd>Esc</kbd>                   |

## Mkdir/Rename InputBox
|Operation          |Key             |
|-------------------|----------------|
|Mkdir/Rename       |<kbd>Enter</kbd>|
|Switch to main page|<kbd>Esc</kbd>  |

## Dialog Boxes
|Operation                          |Key                             |
|-----------------------------------|--------------------------------|
|Switch between textview and buttons|<kbd>Left</kbd>/<kbd>Right</kbd>|
|Scroll in textview                 |<kbd>Up</kbd>/<kbd>Down</kbd>   |
|Select highlighted button          |<kbd>Enter</kbd>                |

## Edit Selection Dialog
|Operation                           |Key              |
|------------------------------------|-----------------|
|Select one item                     |<kbd>,</kbd>     |
|Inverse selection                   |<kbd>a</kbd>     |
|Select all items                    |<kbd>A</kbd>     |
|Switch to filter input              |<kbd>/</kbd>     |
|Switch between filter input and list|<kbd>Tab</kbd>   |
|Save edited list                    |<kbd>Ctrl+s</kbd>|
|Cancel editing list                 |<kbd>Esc</kbd>   |

# Bugs
-  In directories with a huge amount of entries, autocompletion will lag.
   This happens only on the device side (i.e ADB mode), where there is
   significant latency in transferring and processing the directory listing
   to the client.
