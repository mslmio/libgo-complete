# libgo-complete

Shell auto-completion for a Go program, answered by the program itself. No completion script to write, and nothing to keep in step with the command line as it changes.

```bash
go get github.com/mslmio/libgo-complete
```

Zero dependencies outside the standard library. bash, zsh and fish.

## How it works

bash can hand completion to an arbitrary command (`complete -C`): it runs your binary with `COMP_LINE` set to what the user has typed and reads candidates from standard output. So the shell integration is one line in a startup file, and the thing that knows your commands is the thing answering.

## Using it

Describe the command line as a tree, and complete before you parse flags:

```go
var completions = &complete.Command{
    Sub: map[string]*complete.Command{
        "lookup": {
            Flags: map[string]complete.Predictor{
                "--json":  predict.Nothing,
                "--field": predict.Set([]string{"ip", "is_vpn", "vpn.provider"}),
            },
        },
        "database": {
            Sub: map[string]*complete.Command{
                "list":     {},
                "download": {Flags: map[string]complete.Predictor{
                    "--format": predict.Set([]string{"csvgz", "mmdb"}),
                }},
            },
        },
    },
    Flags: map[string]complete.Predictor{"--help": predict.Nothing},
}

func main() {
    completions.Complete("mytool")
    // ... normal argument handling ...
}
```

`Complete` returns immediately unless the shell is asking, so a normal run pays one `getenv`.

Predictors are `predict.Nothing` for a boolean flag, `predict.Set` for a fixed list, `predict.Files`/`predict.Dirs` for paths, and `predict.Func` for anything you compute.

## Installing into the user's shell

```go
install.Install("mytool")    // every shell present on the machine
install.Uninstall("mytool")
install.BashCmd("mytool")    // or print the line for manual installation
```

A startup file is only created for a shell that is actually installed, so a machine without zsh does not grow a `~/.zshrc`. Uninstall writes through a neighbouring file and renames, so an interrupted run cannot leave a truncated startup file.

## Prior art

[posener/complete](https://github.com/posener/complete) established this approach in Go and is worth reading. This is an independent implementation with a smaller surface and no dependencies.

## Licence

MIT.
