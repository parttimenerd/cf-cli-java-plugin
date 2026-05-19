[![REUSE status](https://api.reuse.software/badge/github.com/SAP/cf-cli-java-plugin)](https://api.reuse.software/info/github.com/SAP/cf-cli-java-plugin)
[![Build and Snapshot Release](https://github.com/SAP/cf-cli-java-plugin/actions/workflows/build-and-snapshot.yml/badge.svg)](https://github.com/SAP/cf-cli-java-plugin/actions/workflows/build-and-snapshot.yml)
[![PR Validation](https://github.com/SAP/cf-cli-java-plugin/actions/workflows/pr-validation.yml/badge.svg)](https://github.com/SAP/cf-cli-java-plugin/actions/workflows/pr-validation.yml)

# Cloud Foundry Command Line Java plugin

This plugin for the [Cloud Foundry Command Line](https://github.com/cloudfoundry/cli) provides convenience utilities to
work with Java applications deployed on Cloud Foundry by the [SapMachine](https://sapmachine.io) team.

Currently, it allows you to:

- Trigger and retrieve a heap dump and a thread dump from a Cloud Foundry Java application
- Run jcmd remotely on your application
- Start, stop and retrieve JFR and [async-profiler](https://github.com/jvm-profiling-tools/async-profiler)
  ([SapMachine](https://sapmachine.io) only) profiles from your application
- Run [jstall](https://github.com/parttimenerd/jstall) for one-shot JVM inspection (deadlock detection, hot threads,
  dependency graphs, and more): bundled directly in the plugin, requires Java 17+ locally

## Installation

### Installation via CF Community Repository

Make sure you have the CF Community plugin repository configured (or add it via
`cf add-plugin-repo CF-Community http://plugins.cloudfoundry.org`)

Trigger installation of the plugin via

```sh
cf install-plugin java
```

The releases in the community repository are older than the actual releases on GitHub, that you can install manually, so
we recommend the manual installation.

### Manual Installation

Download the latest release from [GitHub](https://github.com/SAP/cf-cli-java-plugin/releases/latest).

To install a new version of the plugin, run the following:

```sh
# on Mac arm64
cf install-plugin https://github.com/SAP/cf-cli-java-plugin/releases/latest/download/cf-cli-java-plugin-macos-arm64
# on Windows x64
cf install-plugin https://github.com/SAP/cf-cli-java-plugin/releases/latest/download/cf-cli-java-plugin-windows-amd64
# on Linux x64
cf install-plugin https://github.com/SAP/cf-cli-java-plugin/releases/latest/download/cf-cli-java-plugin-linux-amd64
# on Linux arm64
cf install-plugin https://github.com/SAP/cf-cli-java-plugin/releases/latest/download/cf-cli-java-plugin-linux-arm64
```

You can verify that the plugin is successfully installed by looking for `java` in the output of `cf plugins`.

### Manual Installation of Snapshot Release

Download the current snapshot release from [GitHub](https://github.com/SAP/cf-cli-java-plugin/releases/tag/snapshot).
This is intended for experimentation and might fail.

To install a new version of the plugin, run the following:

```sh
# on Mac arm64
cf install-plugin https://github.com/SAP/cf-cli-java-plugin/releases/download/snapshot/cf-cli-java-plugin-macos-arm64
# on Windows x64
cf install-plugin https://github.com/SAP/cf-cli-java-plugin/releases/download/snapshot/cf-cli-java-plugin-windows-amd64
# on Linux x64
cf install-plugin https://github.com/SAP/cf-cli-java-plugin/releases/download/snapshot/cf-cli-java-plugin-linux-amd64
# on Linux arm64
cf install-plugin https://github.com/SAP/cf-cli-java-plugin/releases/download/snapshot/cf-cli-java-plugin-linux-arm64
```

## Usage

### Prerequisites

#### JDK Tools (for `heap-dump` only)

The `heap-dump` command uses `jmap`, which is not shipped by default in the
[Cloud Foundry Java Buildpack](https://github.com/cloudfoundry/java-buildpack). Other commands (`thread-dump`, `jcmd`,
`jfr-*`, `asprof-*`, `status`, `jstall`, `record-status`) use `jcmd` or `asprof`, which are available in SapMachine and
most JDK distributions without extra configuration.

To ensure that `jmap` is available for heap dumps, you can request a full JDK in your application manifest via the
`JBP_CONFIG_OPEN_JDK_JRE` environment variable. This could be done like this:

```yaml
---
applications:
  - name: <APP_NAME>
    memory: 1G
    path: <PATH_TO_BUILD_ARTIFACT>
    buildpack: https://github.com/cloudfoundry/java-buildpack
    env:
      JBP_CONFIG_OPEN_JDK_JRE:
        '{ jre: { repository_root: "https://java-buildpack.cloudfoundry.org/openjdk-jdk/jammy/x86_64", version: 11.+ } }'
      JBP_CONFIG_JAVA_OPTS: "[java_opts: '-XX:+UnlockDiagnosticVMOptions -XX:+DebugNonSafepoints']"
```

`-XX:+UnlockDiagnosticVMOptions -XX:+DebugNonSafepoints` is used to improve profiling accuracy and has no known negative
performance impacts.

Please note that this requires the use of an online buildpack (configured in the `buildpack` property). When system
buildpacks are used, staging will fail with cache issues, because the system buildpacks don’t have the JDK cached.
Please also note that this is not to be considered a recommendation to use a full JDK. It's just one option to get the
tools required for the use of this plugin when you need it, e.g., for troubleshooting. The `version` property is
optional and can be used to request a specific Java version.

#### SSH Access

As it is built directly on `cf ssh`, the `cf java` plugin can work only with Cloud Foundry applications that have
`cf ssh` enabled. To check if your app fulfills the requirements, you can find out by running the
`cf ssh-enabled [app-name]` command. If not enabled yet, run `cf enable-ssh [app-name]`.

**Note:** You must restart your app after enabling SSH access.

In case a proxy server is used, ensure that `cf ssh` is configured accordingly. Refer to the
[official documentation](https://docs.cloudfoundry.org/cf-cli/http-proxy.html#v3-ssh-socks5) of the Cloud Foundry
Command Line for more information. If `cf java` is having issues connecting to your app, chances are the problem is in
the networking issues encountered by `cf ssh`. To verify, run your `cf java` command in "dry-run" mode by adding the
`--dry-run` flag and try to execute the command line that `cf java` gives you back. The plugin now wraps common SSH
failures with user-facing guidance instead of printing the raw `ssh` error verbatim, so running the generated `cf ssh`
command directly is the quickest way to inspect the underlying transport error. If that direct command fails, the issue
is not in `cf java`, but in whatever makes `cf ssh` fail.

### Examples

Getting a heap-dump:

```sh
> cf java heap-dump $APP_NAME
-> ./$APP_NAME-heapdump-$RANDOM.hprof
```

Getting a thread-dump:

```sh
> cf java thread-dump $APP_NAME
...
Full thread dump OpenJDK 64-Bit Server VM ...
...
```

Creating a CPU-time profile via async-profiler:

```sh
> cf java asprof-start-cpu $APP_NAME
Profiling started
# wait some time to gather data
> cf java asprof-stop $APP_NAME
-> ./$APP_NAME-asprof-$RANDOM.jfr
```

Running arbitrary JCMD commands, like `VM.uptime`:

```sh
> cf java jcmd $APP_NAME --args 'VM.uptime'
$TIME s
```

Quick status check of the remote JVM (requires Java 17+ locally):

```sh
> cf java status $APP_NAME
```

Running [JStall](https://github.com/parttimenerd/jstall) for more specific JVM inspection (requires Java 17+ locally):

```sh
# Default: run status analysis with deadlock detection, hot threads, etc.
> cf java jstall $APP_NAME

# Run a specific jstall subcommand
> cf java jstall $APP_NAME --args 'deadlock all'
> cf java jstall $APP_NAME --args 'most-work --dumps 3 all'
> cf java jstall $APP_NAME --args 'flame all'
```

> **Tip:** You can also use JStall directly (without this plugin) via its `--cf` option:
>
> ```sh
> jstall --cf $APP_NAME status all
> ```
>
> This is useful if you want to use a newer JStall version than the one bundled in the plugin. See the
> [JStall README](https://github.com/parttimenerd/jstall) for installation and usage.

Recording JVM diagnostic data for later analysis or sharing:

```sh
# Record all JVM diagnostic data into a zip file (default: APP_NAME-status.zip)
> cf java record-status $APP_NAME

# Record to a specific output file
> cf java record-status $APP_NAME diagnostics.zip

# Record with full data (including expensive jcmd commands, flame graph, and JFR)
> cf java record-status $APP_NAME --full

# Replay the recording locally with jstall
> jstall -f diagnostics.zip status all
> jstall -f diagnostics.zip threads all
```

#### Variable Replacements for JCMD and Asprof Commands

When using `jcmd` and `asprof` commands with the `--args` parameter, the following variables are automatically replaced
in your command strings:

- `@FSPATH`: A writable directory path on the remote container (always set, typically `/tmp/jcmd` or `/tmp/asprof`)
- `@ARGS`: The command arguments you provided via `--args`
- `@APP_NAME`: The name of your Cloud Foundry application
- `@FILE_NAME`: Generated filename for file operations (includes full path with UUID)

Example usage:

```sh
# Create a heap dump in the available directory
cf java jcmd $APP_NAME --args 'GC.heap_dump @FSPATH/my_heap.hprof'

# Use an absolute path instead
cf java jcmd $APP_NAME --args "GC.heap_dump /tmp/absolute_heap.hprof"

# Access the application name in your command
cf java jcmd $APP_NAME --args 'echo "Processing app: @APP_NAME"'
```

**Note**: Variables use the `@` prefix to avoid shell expansion issues. The plugin automatically creates the `@FSPATH`
directory and downloads any files created there to your local directory (unless `--no-download` is used).

### Commands

The following is a list of all available commands (some are SapMachine-specific), generated via `cf java --help`:

<!-- prettier-ignore-start -->
<!-- markdownlint-disable MD036 -->

*Run `cf java --help` to see the full list of commands.*

<!-- markdownlint-enable MD036 -->
<!-- prettier-ignore-end -->

### Security Note on `--args`

The `--args` parameter passes values directly into remote shell commands via `cf ssh`. This is by design to support
shell features like environment variable expansion and piping. **Do not pass untrusted input to `--args`** — treat it
with the same caution as a shell command.

The heap dumps and profiles will be downloaded to a local file automatically (to the current directory by default). Use
`--local-dir` to specify a different download location. To save disk space of the application container, the files are
automatically deleted unless the `--keep` option is set.

Providing `--container-dir` is optional. If specified the plugin will create the heap dump or profile at the given file
path in the application container. Without providing this parameter, the file will be created either at `/tmp` or at the
file path of a file system service if attached to the container.

```shell
cf java [heap-dump|jfr-stop|jfr-dump|asprof-stop] [my-app] --local-dir /local/path [--container-dir /var/fspath]
```

Everything else, like thread dumps, will be output to `std-out`. You may want to redirect the command's output to file,
e.g., by executing:

```shell
cf java thread-dump [my_app] -i [my_instance_index] > thread-dump.txt
```

The `--keep` flag is invalid when invoking non file producing commands. (Unlike with heap dumps, the JVM does not need
to output the thread dump to file before streaming it out.)

## Limitations

Some commands depend on writable filesystem space inside the application container. In particular, `cf java heap-dump`,
`cf java asprof-stop`, and `cf java jfr-stop` first create a file in the container, then stream that file back over SSH,
and finally remove it again unless the `--keep` flag is set.

The available container filesystem space is controlled by the Cloud Foundry landscape configuration and may be limited.
Heap dumps can be large, roughly scaling with heap usage, and profile files can also grow substantially depending on
recording duration and settings. If the container does not have enough free space, the dump or recording cannot be
created and the command will fail.

The plugin uses `pidof java` to locate the target JVM inside the container. If a container runs multiple Java processes,
the plugin may target the wrong one. In such cases, use `jcmd` with an explicit PID obtained via
`cf ssh APP -c 'ps aux | grep java'`.

The plugin is also constrained by limitations in the current `cf-cli` plugin framework:

- `CF_TRACE=true` will break file-producing commands (`heap-dump`, `jfr-stop`, `asprof-stop`, `jcmd`). Disable
  `CF_TRACE` before using these commands, or use `--dry-run` and run the generated command directly.
- There is no distinction between `stdout` and `stderr` output from the underlying `cf ssh` command (see
  [this issue on the `cf-cli` project](https://github.com/cloudfoundry/cli/issues/1074))
  - `cf java` will still usually exit with status code `1` when the underlying `cf ssh` command fails
  - If you need separate `stdout` and `stderr`, run the plugin in dry-run mode (`--dry-run`) and execute the generated
    command directly

### Known Command Limitations

#### jstall Flame Graph May Fail in Containerized Environments

The `jstall flame` command may fail with an error like:

```text
Error: profiling was skipped: profiling-failed
jstall execution failed: exit status 1
```

**Reason:** Flame graph generation depends on low-level profiling capabilities such as `perf` events. These are often
restricted in containerized environments for security reasons.

**Workarounds:**

1. Use async-profiler directly for CPU profiling:

   ```bash
   cf java asprof-start-cpu $APP_NAME
   # ... wait for profiling ...
   cf java asprof-stop $APP_NAME
   ```

2. Use other jstall commands that don't require perf:

   ```bash
   cf java jstall $APP_NAME --args 'status all'      # JVM status & diagnostics
   cf java jstall $APP_NAME --args 'deadlock all'    # Deadlock detection
   cf java jstall $APP_NAME --args 'threads all'     # Thread information
   ```

3. Record diagnostic data for later analysis:

   ```bash
   cf java record-status $APP_NAME diagnostics.zip
   # Then replay locally
   jstall -f diagnostics.zip status all
   ```

#### SSH Connection Failures

When the plugin cannot establish SSH connectivity, it reports a categorized error message with likely causes and
suggested next steps instead of only printing the raw `cf ssh` transport error. A typical message looks like:

```text
Cannot connect to app 'APP_NAME' via SSH.
Possible causes and solutions:
1. SSH may not be enabled on the application. Try:
   cf enable-ssh APP_NAME
   cf restart APP_NAME
2. Check your network connection and firewall settings.
3. Verify the application is running: cf app APP_NAME
```

**Common causes and fixes:**

| Error                                 | Likely Cause                   | Solution                                                    |
| ------------------------------------- | ------------------------------ | ----------------------------------------------------------- |
| `connection refused` or `not enabled` | SSH not enabled on application | `cf enable-ssh APP_NAME && cf restart APP_NAME`             |
| `connection reset`                    | Network interruption           | Retry the command; check internet connection                |
| `timeout`                             | Network unreachable            | Check firewall/proxy settings; verify platform connectivity |
| `Permission denied`                   | Authentication failed          | `cf logout && cf login` with correct credentials            |

**Debugging:** If you need the original SSH transport error from Cloud Foundry, run `cf ssh APP_NAME -c 'echo ok'`
directly.

## Side-effects on the running instance

Creating dumps and profile recordings consumes container filesystem space. If too much space is used, other operations
inside the container (for example, writing temporary files) may fail, which can lead to unexpected application errors.

### Thread-Dumps

Capturing a thread dump with `cf java` usually has low overhead on the JVM, unless the process has a very large number
of threads.

### Heap-Dumps

Heap dumps require more care. Triggering a heap dump typically causes a full GC, during which the JVM can become
temporarily unresponsive. The overall impact depends on heap size (larger heaps usually take longer), GC behavior, and
especially whether the container is swapping memory to disk. Swapping is generally very costly for JVM performance.

Because Cloud Foundry cells can be overcommitted, a container may begin swapping during dump generation (or may already
be swapping before it starts). In stressed conditions, generating a heap dump can therefore further degrade application
performance.

### Profiles

Profiling introduces overhead that depends on the selected mode and settings, but default configurations are usually
designed to keep that overhead moderate.

## Development

### Quick Start

```bash
# Setup environment and build
./setup-dev-env.sh
make build

# Run all quality checks and tests
./scripts/lint-all.sh ci

# Auto-fix formatting before commit
./scripts/lint-all.sh fix
```

### Build Configuration

**JStall Version**: By default, the build downloads the latest stable JStall release. To test with the latest
development build from GitHub Actions instead, use:

```bash
JSTALL_DEV=1 make build

# Or download a specific GitHub Actions run by ID
JSTALL_DEV=<run-id> make build
```

This pulls the latest JStall build directly from the GitHub Actions artifacts instead of the released version.

### Testing

**Python Tests**: Modern pytest-based test suite.

```bash
cd test && ./setup.sh && ./test.py all
```

### Test Suite Resumption

The Python test runner in `test/` supports resuming tests from any point using the `--start-with` option:

```bash
./test.py --start-with TestClass::test_method all  # Start with a specific test (inclusive)
```

This is useful for long test suites or after interruptions. See `test/README.md` for more details.

### Code Quality

Centralized linting scripts:

```bash
./scripts/lint-all.sh check    # Quality check
./scripts/lint-all.sh fix      # Auto-fix formatting
./scripts/lint-all.sh ci       # CI validation
```

### CI/CD

- Multi-platform builds (Linux, macOS, Windows)
- Automated linting and testing on PRs
- Pre-commit hooks with auto-formatting

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via
[GitHub issues](https://github.com/SAP/cf-cli-java-plugin/issues). Contribution and feedback are encouraged and always
welcome. Just be aware that this plugin is limited in scope to keep it maintainable. For more information about how to
contribute, the project structure, as well as additional contribution information, see our
[Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure

If you find any bug that may be a security problem, please follow our instructions at
[in our security policy](https://github.com/SAP/cf-cli-java-plugin/security/policy) on how to report it. Please do not
create GitHub issues for security-related doubts or problems.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a detailed list of changes.

## License

Copyright 2017 - 2026 SAP SE or an SAP affiliate company and contributors. Please see our LICENSE for copyright and
license information.
