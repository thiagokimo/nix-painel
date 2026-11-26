# painel

A terminal home screen: the tools, scripts, projects and links I use every day,
in one place. Type a name or alias and press enter.

- **tools**: nvim, yazi, btop… painel hands them the terminal and comes back
  when they exit.
- **scripts**: sysup, flakeup… with `pause`, the output stays on screen until
  you press enter.
- **projects**: every folder in the roots, with its git branch. Enter opens
  nvim there, `y` opens yazi.
- **links**: open in the browser.

## Keys

painel starts in the prompt.

| prompt              |                                     |
| ------------------- | ----------------------------------- |
| type                | filter everything, best match first |
| enter               | run the picked match; a URL opens; anything else is a web search |
| ↑ ↓ / ctrl+p ctrl+n | pick another match                  |
| tab                 | complete the name                   |
| esc                 | clear, then switch to navigation    |

| navigation  |                            |
| ----------- | -------------------------- |
| h j k l     | move between and in panels |
| enter       | run                        |
| y           | open a project in yazi     |
| r           | find projects again        |
| / i :       | back to the prompt         |
| q / ctrl+c  | quit                       |

## Config

`$XDG_CONFIG_HOME/painel/config.json`, or `painel -config <file>`. Anything left
out keeps its default; a list in the file replaces the default list. See
[example/config.json](example/config.json).

| key        |                                                              |
| ---------- | ------------------------------------------------------------ |
| `user`, `host` | shown in the header; default to the current ones          |
| `theme`    | `#rrggbb` colors: fg, fg_bright, muted, border, selection, accent, red, orange, yellow, green, aqua, blue |
| `tools`, `scripts` | `{name, alias?, desc?, cmd: [...], pause?}`          |
| `projects` | `roots`, `extra` folders, and the `open`/`alt` commands, run inside the project |
| `links`    | groups: `{name, links: [{name, alias?, url}]}`               |
| `browser`  | command that opens a URL, given as its last argument         |
| `search`   | URL prefix for text that matches nothing; `""` turns it off  |

Aliases must be unique across everything.

## Nix

```nix
inputs.painel.url = "github:thiagokimo/painel";

# Home Manager
imports = [inputs.painel.homeManagerModules.default];
programs.painel = {
  enable = true;
  settings = { /* same keys as config.json */ };
};
```

`nix run`, `nix develop`, and `nix flake check` (builds and runs the tests).
