### Planned Features

- [ ] Command to capture the current home dir layout and save it to a config file
- [ ] Dirs marked as volatile, that keep the files in memory only (so they disappear on reboot)
- [ ] make tui be able to manage tags of already sorted files / dirs
- [ ] display available tags in tui
- [ ] only display compatible tags in tui
- [ ] check tag compatibility with the file and dir the tags fit to (e.g., check that image tag isn't used with a text file because force mime type is set)
- [ ] config option to automatically sort files when x% sure that it guesses the correct tags
- [ ] for dirs check if a suggestion can be made based on the dirs content
- [ ] in the tui move to the next entry as soon as tags are enough to infer the exact dir it has to go to (remove the auto move from the suggestion thing when that is implemented)
- [ ] options in nix config to inject into shell to auto launch manager when cd-ing into inbox dir and thing that shows in the terminal how many inbox items there currently are 
- [ ] inbox waybar widget
- [ ] when tags are ambiguous give the use the option to display the files in both dirs 