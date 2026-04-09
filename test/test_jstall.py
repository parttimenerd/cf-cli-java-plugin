"""
JStall integration tests for CF Java Plugin.

Run with:
    pytest test_jstall.py -v                     # All jstall tests
    pytest test_jstall.py::TestJStallDryRun -v    # Only dry-run tests
    pytest test_jstall.py::TestJStallErrors -v    # Only error-handling tests
    pytest test_jstall.py::TestJStallExecution -v # Only live execution tests
    pytest test_jstall.py::TestRecordStatusDryRun -v  # Only record-status dry-run tests
    pytest test_jstall.py::TestRecordStatusErrors -v  # Only record-status error tests

Tests require Java 17+ to be installed locally.
JStall is a JVM inspection tool bundled as jstall-minimal.jar in the plugin binary.
See https://github.com/parttimenerd/jstall
"""

import os

import pytest

from framework.decorators import test
from framework.runner import TestBase


def java17_available():
    """Check if Java 17+ is available locally."""
    import subprocess

    try:
        result = subprocess.run(["java", "-version"], capture_output=True, text=True, timeout=10)
        output = result.stderr + result.stdout
        # Parse version from output like: openjdk version "21.0.1" or java version "17.0.5"
        import re

        match = re.search(r'"(\d+)', output)
        if match:
            major = int(match.group(1))
            return major >= 17
    except (FileNotFoundError, subprocess.TimeoutExpired, ValueError):
        pass
    return False


requires_java17 = pytest.mark.skipif(not java17_available(), reason="Java 17+ not available locally")


class TestJStallDryRun(TestBase):
    """Dry-run tests for JStall — no Java or CF connectivity required."""

    @test(no_restart=True)
    def test_dry_run_default(self, t, app):
        """Test jstall dry run shows the java command with --ssh and no default subcommand."""
        result = t.run(f"jstall {app} --dry-run").should_succeed()
        result.should_contain("-jar")
        result.should_contain("jstall-minimal.jar")
        result.should_contain("--ssh")
        result.should_contain(f"cf ssh {app}")
        result.should_not_contain("status all")

    @test(no_restart=True)
    def test_dry_run_with_args(self, t, app):
        """Test jstall dry run with custom subcommand via --args."""
        result = t.run(f"jstall {app} --dry-run --args 'deadlock'").should_succeed()
        result.should_contain("--ssh")
        result.should_contain(f"cf ssh {app}")
        result.should_contain("deadlock")
        result.should_not_contain("status")

    @test(no_restart=True)
    def test_dry_run_with_complex_args(self, t, app):
        """Test jstall dry run with multi-word args."""
        result = t.run(f"jstall {app} --dry-run --args 'most-work --dumps 3'").should_succeed()
        result.should_contain("most-work")
        result.should_contain("--dumps")
        result.should_contain("3")

    @test(no_restart=True)
    def test_dry_run_with_instance_index(self, t, app):
        """Test jstall dry run with --app-instance-index includes it in --ssh."""
        result = t.run(f"jstall {app} --dry-run --app-instance-index 2").should_succeed()
        result.should_contain("--ssh")
        result.should_contain("--app-instance-index")
        result.should_contain("2")

    @test(no_restart=True)
    def test_dry_run_instance_index_zero(self, t, app):
        """Test jstall dry run with --app-instance-index 0 omits the index (default)."""
        result = t.run(f"jstall {app} --dry-run --app-instance-index 0").should_succeed()
        result.should_contain("--ssh")
        result.should_not_contain("--app-instance-index")

    @test(no_restart=True)
    def test_dry_run_command_structure(self, t, app):
        """Test that the dry-run output has the correct argument ordering: java -jar JAR --ssh CMD."""
        result = t.run(f"jstall {app} --dry-run").should_succeed()
        # The output should be a single command line ending after -c with no default subcommand
        result.should_match(r"java.*-jar.*jstall-minimal\.jar.*--ssh.*cf ssh")

    @test(no_restart=True)
    def test_dry_run_ssh_command_has_dash_c(self, t, app):
        """Test that --ssh argument ends with '-c' for cf ssh."""
        result = t.run(f"jstall {app} --dry-run").should_succeed()
        result.should_contain("-c")

    @test(no_restart=True)
    def test_no_files_created(self, t, app):
        """Test that jstall dry-run creates no local files."""
        t.run(f"jstall {app} --dry-run").should_succeed().no_files()

    @test(no_restart=True)
    def test_verbose_dry_run(self, t, app):
        """Test jstall verbose dry run shows internal details."""
        result = t.run(f"jstall {app} --dry-run --verbose").should_succeed()
        result.should_contain("[VERBOSE]")

    @test(no_restart=True)
    def test_verbose_shows_java_path(self, t, app):
        """Test that verbose output mentions the Java path found."""
        result = t.run(f"jstall {app} --dry-run --verbose").should_succeed()
        result.should_contain("Found Java 17+")

    @test(no_restart=True)
    def test_verbose_shows_jar_path(self, t, app):
        """Test that verbose output mentions the JAR path."""
        result = t.run(f"jstall {app} --dry-run --verbose").should_succeed()
        result.should_contain("JStall JAR at")

    @test(no_restart=True)
    def test_verbose_shows_full_command(self, t, app):
        """Test that verbose output shows the constructed command line."""
        result = t.run(f"jstall {app} --dry-run --verbose").should_succeed()
        result.should_contain("JStall command:")

    @test(no_restart=True)
    def test_dry_run_with_deadlock_subcommand(self, t, app):
        """Test jstall dry run with deadlock detection subcommand."""
        result = t.run(f"jstall {app} --dry-run --args 'deadlock'").should_succeed()
        result.should_contain("deadlock")
        result.should_not_contain("status")
        result.should_not_contain("all")

    @test(no_restart=True)
    def test_dry_run_with_hot_threads(self, t, app):
        """Test jstall dry run with hot-threads subcommand."""
        result = t.run(f"jstall {app} --dry-run --args 'hot-threads'").should_succeed()
        result.should_contain("hot-threads")

    @test(no_restart=True)
    def test_dry_run_with_thread_dump(self, t, app):
        """Test jstall dry run with thread-dump subcommand."""
        result = t.run(f"jstall {app} --dry-run --args 'thread-dump'").should_succeed()
        result.should_contain("thread-dump")

    @test(no_restart=True)
    def test_dry_run_with_vm_vitals(self, t, app):
        """Test jstall dry run with vm-vitals subcommand."""
        result = t.run(f"jstall {app} --dry-run --args 'vm-vitals'").should_succeed()
        result.should_contain("vm-vitals")

    @test(no_restart=True)
    def test_dry_run_with_list(self, t, app):
        """Test jstall dry run with list subcommand."""
        result = t.run(f"jstall {app} --dry-run --args 'list'").should_succeed()
        result.should_contain("list")
        result.should_not_contain("status")

    @test(no_restart=True)
    def test_dry_run_with_args_containing_flags(self, t, app):
        """Test jstall dry run with subcommand args that have their own flags."""
        result = t.run(f"jstall {app} --dry-run --args 'hot-threads --top 5 --interval 2'").should_succeed()
        result.should_contain("hot-threads")
        result.should_contain("--top")
        result.should_contain("5")
        result.should_contain("--interval")
        result.should_contain("2")

    @test(no_restart=True)
    def test_dry_run_with_instance_index_large(self, t, app):
        """Test jstall dry run with a high instance index."""
        result = t.run(f"jstall {app} --dry-run --app-instance-index 99").should_succeed()
        result.should_contain("--app-instance-index")
        result.should_contain("99")

    @test(no_restart=True)
    def test_dry_run_jstall_with_full_flag_rejected(self, t, app):
        """Test jstall dry run with --full is rejected (not supported for jstall passthrough)."""
        t.run(f"jstall {app} --dry-run --full").should_fail().should_contain(
            'The flag "full" is not supported for jstall'
        )

    @test(no_restart=True)
    def test_dry_run_status_with_full_flag(self, t, app):
        """Test status dry run with --full appends --full to the jstall command."""
        result = t.run(f"status {app} --dry-run --full").should_succeed()
        result.should_contain("status")
        result.should_contain("all")
        result.should_contain("--full")

    @test(no_restart=True)
    def test_dry_run_status_without_full(self, t, app):
        """Test status dry run without --full does not include --full."""
        result = t.run(f"status {app} --dry-run").should_succeed()
        result.should_contain("status")
        result.should_contain("all")
        result.should_not_contain("--full")


class TestJStallErrors(TestBase):
    """Error-handling and input-validation tests for JStall."""

    @test(no_restart=True)
    def test_file_flags_rejected_keep(self, t, app):
        """Test that --keep flag is rejected for jstall."""
        t.run(f"jstall {app} --keep").should_fail().should_contain('The flag "keep" is not supported for jstall')

    @test(no_restart=True)
    def test_file_flags_rejected_no_download(self, t, app):
        """Test that --no-download flag is rejected for jstall."""
        t.run(f"jstall {app} --no-download").should_fail().should_contain(
            'The flag "no-download" is not supported for jstall'
        )

    @test(no_restart=True)
    def test_file_flags_rejected_container_dir(self, t, app):
        """Test that --container-dir flag is rejected for jstall."""
        t.run(f"jstall {app} --container-dir /tmp").should_fail().should_contain(
            'The flag "container-dir" is not supported for jstall'
        )

    @test(no_restart=True)
    def test_file_flags_rejected_local_dir(self, t, app):
        """Test that --local-dir flag is rejected for jstall."""
        t.run(f"jstall {app} --local-dir /tmp").should_fail().should_contain(
            'The flag "local-dir" is not supported for jstall'
        )

    @test(no_restart=True)
    def test_no_app_name(self, t, app):
        """Test that omitting the app name produces an error."""
        t.run("jstall").should_fail().should_contain("No application name provided")

    @test(no_restart=True)
    def test_too_many_arguments(self, t, app):
        """Test that extra positional arguments produce an error."""
        t.run(f"jstall {app} extraarg").should_fail().should_contain("Too many arguments provided")

    @test(no_restart=True)
    def test_invalid_flag(self, t, app):
        """Test that an unknown flag produces an error."""
        t.run(f"jstall {app} --not-a-real-flag").should_fail().should_contain("Error while parsing command arguments")

    @test(no_restart=True)
    def test_negative_app_instance_index(self, t, app):
        """Test that a negative instance index is rejected."""
        t.run(f"jstall {app} --app-instance-index -1").should_fail().should_contain(
            "Invalid application instance index -1, must be >= 0"
        )

    @test(no_restart=True)
    def test_non_numeric_app_instance_index(self, t, app):
        """Test that a non-numeric instance index is rejected."""
        t.run(f"jstall {app} --app-instance-index abc").should_fail().should_contain(
            "Error while parsing command arguments"
        )

    @test(no_restart=True)
    def test_help_output(self, t, app):
        """Test that --help shows usage information."""
        t.run(f"jstall {app} --help").should_succeed().should_contain_help()


class TestFullFlagRejection(TestBase):
    """Tests that --full is rejected for non-local commands."""

    @test(no_restart=True)
    def test_full_rejected_for_heap_dump(self, t, app):
        """Test that --full is rejected for heap-dump (non-local command)."""
        t.run(f"heap-dump {app} --full").should_fail().should_contain('The flag "full" is not supported for heap-dump')

    @test(no_restart=True)
    def test_full_rejected_for_thread_dump(self, t, app):
        """Test that --full is rejected for thread-dump (non-local command)."""
        t.run(f"thread-dump {app} --full").should_fail().should_contain(
            'The flag "full" is not supported for thread-dump'
        )


class TestJStallExecution(TestBase):
    """Live execution tests for JStall — require Java 17+ and CF connectivity."""

    @requires_java17
    @test(no_restart=True)
    def test_status_execution(self, t, app):
        """Test actual jstall status execution against the app."""
        t.run(f"jstall {app}").should_succeed()

    @requires_java17
    @test(no_restart=True)
    def test_status_command_not_local_pid_output(self, t, app):
        """Test status command does not fall back to local PID scanning output."""
        t.run(f"status {app}").should_succeed().should_not_contain("====== PID ")

    @requires_java17
    @test(no_restart=True)
    def test_list_execution(self, t, app):
        """Test jstall list subcommand via --args."""
        t.run(f"jstall {app} --args 'list'").should_succeed()

    @requires_java17
    @test(no_restart=True)
    def test_vm_vitals_via_jstall(self, t, app):
        """Test jstall vm-vitals subcommand."""
        t.run(f"jstall {app} --args 'vm-vitals'").should_succeed()

    @requires_java17
    @test(no_restart=True)
    def test_deadlock_detection(self, t, app):
        """Test jstall deadlock detection subcommand."""
        t.run(f"jstall {app} --args 'deadlock'").should_succeed()

    @requires_java17
    @test(no_restart=True)
    def test_hot_threads(self, t, app):
        """Test jstall hot-threads subcommand."""
        t.run(f"jstall {app} --args 'hot-threads'").should_succeed()

    @requires_java17
    @test(no_restart=True)
    def test_thread_dump_via_jstall(self, t, app):
        """Test jstall thread-dump subcommand."""
        t.run(f"jstall {app} --args 'thread-dump'").should_succeed()

    @requires_java17
    @test(no_restart=True)
    def test_status_with_instance_zero(self, t, app):
        """Test jstall with explicit instance index 0."""
        t.run(f"jstall {app} --app-instance-index 0").should_succeed()

    @requires_java17
    @test(no_restart=True)
    def test_wrong_instance_index(self, t, app):
        """Test jstall with wrong instance index fails gracefully."""
        t.run(f"jstall {app} --app-instance-index 99").should_fail()


class TestRecordStatusDryRun(TestBase):
    """Dry-run tests for the record-status command."""

    @test(no_restart=True)
    def test_dry_run_with_output_file(self, t, app):
        """Test record-status dry run with an explicit output file as trailing arg."""
        result = t.run(f"record-status {app} out.zip --dry-run").should_succeed()
        result.should_contain("record")
        result.should_contain("--output")
        result.should_contain("out.zip")
        result.should_contain("--ssh")
        result.should_contain(f"cf ssh {app}")
        result.should_contain("jstall-minimal.jar")

    @test(no_restart=True)
    def test_dry_run_default_output_file(self, t, app):
        """Test record-status dry run without output file uses APP_NAME-status.zip default."""
        result = t.run(f"record-status {app} --dry-run").should_succeed()
        result.should_contain("record")
        result.should_contain("--output")
        result.should_contain(f"{app}-status.zip")

    @test(no_restart=True)
    def test_dry_run_with_args_output_file(self, t, app):
        """Test record-status dry run with output file via --args."""
        result = t.run(f"record-status {app} --dry-run --args 'custom-output.zip'").should_succeed()
        result.should_contain("record")
        result.should_contain("--output")
        result.should_contain("custom-output.zip")

    @test(no_restart=True)
    def test_dry_run_with_instance_index(self, t, app):
        """Test record-status dry run with --app-instance-index."""
        result = t.run(f"record-status {app} out.zip --dry-run --app-instance-index 2").should_succeed()
        result.should_contain("--app-instance-index")
        result.should_contain("2")
        result.should_contain("record")
        result.should_contain("--output")
        result.should_contain("out.zip")

    @test(no_restart=True)
    def test_dry_run_no_files_created(self, t, app):
        """Test that record-status dry run creates no local files."""
        t.run(f"record-status {app} out.zip --dry-run").should_succeed().no_files()

    @test(no_restart=True)
    def test_dry_run_verbose(self, t, app):
        """Test record-status verbose dry run shows internal details."""
        result = t.run(f"record-status {app} out.zip --dry-run --verbose").should_succeed()
        result.should_contain("[VERBOSE]")
        result.should_contain("JStall command:")

    @test(no_restart=True)
    def test_dry_run_command_structure(self, t, app):
        """Test record-status dry run has correct argument ordering."""
        result = t.run(f"record-status {app} out.zip --dry-run").should_succeed()
        result.should_match(r"java.*-jar.*jstall-minimal\.jar.*--ssh.*cf ssh.*record\s+all\s+--output\s+out\.zip")

    @test(no_restart=True)
    def test_does_not_contain_status_all(self, t, app):
        """Test record-status does NOT fall back to 'status all' default."""
        result = t.run(f"record-status {app} --dry-run").should_succeed()
        result.should_contain("record")
        result.should_contain("--output")
        result.should_not_contain("status all")

    @test(no_restart=True)
    def test_dry_run_contains_all_target(self, t, app):
        """Test record-status dry run includes 'all' as the jstall record target."""
        result = t.run(f"record-status {app} --dry-run").should_succeed()
        result.should_match(r"record\s+all\s+--output")

    @test(no_restart=True)
    def test_dry_run_with_full_flag(self, t, app):
        """Test record-status dry run with --full flag."""
        result = t.run(f"record-status {app} out.zip --dry-run --full").should_succeed()
        result.should_contain("record")
        result.should_contain("--output")
        result.should_contain("out.zip")
        result.should_contain("--full")

    @test(no_restart=True)
    def test_dry_run_with_full_via_args(self, t, app):
        """Test record-status dry run with --full passed via --args."""
        result = t.run(f"record-status {app} out.zip --dry-run --args '--full'").should_succeed()
        result.should_contain("--full")


class TestRecordStatusErrors(TestBase):
    """Error-handling tests for the record-status command."""

    @test(no_restart=True)
    def test_file_flags_rejected_keep(self, t, app):
        """Test that --keep is rejected for record-status."""
        t.run(f"record-status {app} --keep").should_fail().should_contain(
            'The flag "keep" is not supported for record-status'
        )

    @test(no_restart=True)
    def test_file_flags_rejected_no_download(self, t, app):
        """Test that --no-download is rejected for record-status."""
        t.run(f"record-status {app} --no-download").should_fail().should_contain(
            'The flag "no-download" is not supported for record-status'
        )

    @test(no_restart=True)
    def test_file_flags_rejected_container_dir(self, t, app):
        """Test that --container-dir is rejected for record-status."""
        t.run(f"record-status {app} --container-dir /tmp").should_fail().should_contain(
            'The flag "container-dir" is not supported for record-status'
        )

    @test(no_restart=True)
    def test_file_flags_rejected_local_dir(self, t, app):
        """Test that --local-dir is rejected for record-status."""
        t.run(f"record-status {app} --local-dir /tmp").should_fail().should_contain(
            'The flag "local-dir" is not supported for record-status'
        )

    @test(no_restart=True)
    def test_no_app_name(self, t, app):
        """Test that omitting the app name produces an error."""
        t.run("record-status").should_fail().should_contain("No application name provided")

    @test(no_restart=True)
    def test_invalid_flag(self, t, app):
        """Test that an unknown flag produces an error."""
        t.run(f"record-status {app} --not-a-real-flag").should_fail().should_contain(
            "Error while parsing command arguments"
        )

    @test(no_restart=True)
    def test_negative_app_instance_index(self, t, app):
        """Test that a negative instance index is rejected."""
        t.run(f"record-status {app} --app-instance-index -1").should_fail().should_contain(
            "Invalid application instance index -1, must be >= 0"
        )

    @test(no_restart=True)
    def test_help_output(self, t, app):
        """Test that --help shows usage information."""
        t.run(f"record-status {app} --help").should_succeed().should_contain_help()


class TestRecordStatusExecution(TestBase):
    """Live execution tests for record-status — require Java 17+ and CF connectivity."""

    @requires_java17
    @test(no_restart=True)
    def test_record_status_with_output_file(self, t, app):
        """Test actual record-status execution with an output file."""
        output_file = "test-output.zip"
        try:
            t.run(f"record-status {app} {output_file}").should_succeed()
            assert os.path.exists(output_file), f"Expected output file '{output_file}' to exist"
            assert os.path.getsize(output_file) > 1024, f"Expected output file '{output_file}' to be larger than 1KB"
        finally:
            if os.path.exists(output_file):
                os.remove(output_file)

    @requires_java17
    @test(no_restart=True)
    def test_record_status_default_output(self, t, app):
        """Test actual record-status execution with default output file."""
        t.run(f"record-status {app}").should_succeed()
