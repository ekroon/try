# try

A fast, interactive project selector for the command line. Quickly navigate between your projects with substring search and create new ones on the fly.

Inspired by [tobi/try](https://github.com/tobi/try).

## Installation

### Option 1: Manual installation

1. Build the binary:
   ```bash
   go build -o try main.go
   ```

2. Add the shell integration to your `.bashrc` or `.zshrc`:
   ```bash
   # Add try to your PATH first
   export PATH="$PATH:/path/to/try"
   
   # Initialize the shell function
   eval "$(try init)"
   ```

### Option 2: Using mise

Add to your `.zshrc`:
```bash
eval "$(mise x ubi:ekroon/try -- try init)"
```

## Usage

```bash
try  # Launch the interactive project selector
```

### Keyboard shortcuts:
- `Ctrl+K/Ctrl+J` - Navigate up/down
- `Enter` - Select project and cd into it
- `Ctrl+N` - Create new project with current search term
- `Ctrl+Q` - Quit
- Type to search projects

## Configuration

Set a custom projects directory:
```bash
export TRY_PROJECTS_DIR="/path/to/your/projects"
```

Default: `~/projects`

## Project layout

New projects created from the CLI use a date bucket layout:

```text
~/projects/YYYY-MM-DD/project-name
```

Existing legacy directories that use a date-prefixed name are still supported and remain selectable:

```text
~/projects/YYYY-MM-DD-project-name
```

## Features

- 🔍 Substring search through project directories
- ⚡ Fast navigation with keyboard shortcuts  
- 📁 Create new projects in `YYYY-MM-DD/project` date buckets
- ♻️ Continue supporting existing `YYYY-MM-DD-project` directories
- 🎨 Beautiful terminal interface
- 🐚 Shell integration that actually changes your directory

## TODO

- [ ] Implement actual fuzzy search for better matching
