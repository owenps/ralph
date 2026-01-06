# Ralph TUI

> [!WARNING]
> Ralph is still a work-in-progress, so far it's just a task manager.
> Stay tuned for updates!

A TUI for planning and running claude code on a autonmous loop based on the
[Ralph Wiggem](https://ghuntley.com/ralph/) technique.

![ralph-wiggem](https://media1.tenor.com/m/u1_XOvKApKgAAAAC/ralph-wiggum.gif)

## What Ralph Can Do

Ralph is built to help chew through your development backlog and never ending list
of TODOs. Run Ralph when you are taking a break or while away from your desk!

The basic flow is as [simple as the man himself](https://www.youtube.com/watch?v=wUpe0Q1HnR4)!

1. [✔︎] **Create Tasks**: Create small digestible _task_ that you never got around
   to completing.
1. [~] **Create Sprints**: Select a set of tasks into a _sprint_.
1. [~] **Run Ralph Loop**: Execute the sprint which runs claude in a loop for a maximum
   number of iterations.

Ralph tracks each task as a individual git commit so you can review the changes
and polish before creating a PR. As Ralph is working, it'll update it progress
in `progress.txt` so the next iteration (or even you) can understand the full
scope of the current work.

## Installation

Smooth installation is still work in progress. For now pull down the repo and build
the executable.

```sh
go build
./ralph
```

Once you've setup the global config, you can navigate to any other directory with
Ralph and run initialize the project level config.

```sh
./ralph init
```

## Data

- Ralph creates one user config under `~/.config/ralph/`
- Project-level data (tasks, sprints, etc.) can be found at `.ralph/` in your project
  directory
