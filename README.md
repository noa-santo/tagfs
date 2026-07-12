# tagfs

A home directory manager that forces you to keep your files tidy and organized through semantic tags.
I created this because I liked how I can configure everything in my os via a nix config, except one thing:
my home directory.
Often when I just quickly test something, I create it in my home dir, and then I forget to move it to the right dir.
So the most sensible thing was obviously to create a custom FUSE (Filesystem in Userspace) that would automatically move
the files to the right dir based on the tags (instead of just moving them to the right dir :p).
Making my home dir a FUSE is also kinda cool because that lets me program special behavior for stuff that I might wanna
do in the future.

Currently, it's a work in progress.
Data loss is possible (and currently tbh pretty likely).

### Features

- [x] Configurable home dir layout with nix home manager
- [x] Automatically move files into the "Inbox" dir when created in the home dir
- [x] Passthrough dirs for e.g., .config and .cache
- [x] Tag suggestion based on mime type and file name patterns
- [x] Modern TUI for managing inbox items
- [x] Configurable directory rules (e.g., only allow images in the pictures dir): mime type, file name patterns, allow
  subdir creation, allow file creation

### Planned Features

- [ ] Command to capture the current home dir layout and save it to a config file
- [ ] Dirs marked as volatile, that keep the files in memory only (so they disappear on reboot)
- [ ] overwrite tag to ignore dir rules
- [ ] make tui be able to manage tags of already sorted files / dirs
- [ ] display available tags in tui
- [ ] only display compatible tags in tui
- [ ] check tag compatibility with the file and dir the tags fit to (e.g., check that image tag isn't used with a text file because force mime type is set)
- [ ] config option to automatically sort files when x% sure that it guesses the correct tags
- [ ] for dirs check if a suggestion can be made based on the dirs content
- [ ] when tags are ambiguous give the use the option to display the files in both dirs 
- [ ] backup db data for files into it's xattr so that the db can be restored when lost
- [ ] cache virtual path for faster lookup
- [ ] performance improvements

### Features that would be cool to have but idk if i'll ever implement them

- [ ] plugins for the fuse to add aditional functionality
- [ ] inbox waybar widget
- [ ] options in nix config to inject into shell to auto launch manager when cd-ing into inbox dir and thing that shows
  in the terminal how many inbox items there currently are
- [ ] command to move files from the flat store back into a real structure in case anyone wants to stop using tagfs

### How does it work?

All files and dirs are stored structure-less in a store path.
What tags a dir belongs to is stored in a sqlite database.
The fuse then dynamically resolves the contents of a dir based on the tags.
For that a file has to have enough tags to unambiguously infer the dir it belongs to.

### Screenshots

<img width="3836" height="1922" alt="image" src="https://github.com/user-attachments/assets/8cad5ce9-b1ef-414a-b2e2-b554fd6dcd37" />
