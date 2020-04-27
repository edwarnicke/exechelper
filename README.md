exechelper is a simple wrapper around standard go 'exec' that tries to follow the spirit of its API closely

Its main features are:

1. It allows you to pass strings containing the command you want executed instead of an array of args.
2. It's Start() returns an errCh that will get zero or one errors, and be closed after the command has finished running
3. You can use the WithXYZ pattern to do things like customize things like Stdout, Stdin, StdError, Dir, and Env variables

Run Examples:

```go
if err := exechelper.Run("go list -m");err != nil {...}

if err := exechelper.Run("go list -m",WithEnv(os.Environs()));err != nil {...}

if err := exechelper.Run("go list -m",WithDir(dir)); err != nil {...}

outputBuffer := bytes.NewBuffer([]byte{})
errBuffer := bytes.NewBuffer([]bytes{})
if err := exechelper.Run("go list -m",WithStdout(outputBuffer),WithStderr(errBuffer));err != nil {...}
```

Start Examples

```go
ctx,cancel := context.WithCancel(context.Background())
errCh := exechelper.Start(startContext,"spire-server run",WithContext(ctx))

// cancel() - will stop the running cmd if its still running
// <-errCh - provides any error from the Start.  Will always have zero or one err, so <-errCh will block until Start has finished
//           For non-zero Exit code will return a exec.ExitError
//           Note: cancel() will almost always result in a non-zero exit code
```

Similarly, exechelper.Output(...), exechelper.CombinedOutput(...) are provided.

