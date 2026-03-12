# gitex

A cross-platform CLI tool that exports GitLab issues as local markdown documents with images and attachments.

## Features

- **Export issues** — single issue or all issues from a project
- **Image download** — images from descriptions and comments are saved locally, with paths rewritten in the markdown
- **Document attachments** — PDFs, Office documents, ZIP archives, `.drawio` files, etc. are downloaded to an `appendix/` directory
- **Draw.io rendering** — `.drawio` diagrams are automatically rendered to PNG (if the draw.io CLI is available) and embedded in the markdown
- **Filtering** — filter issues by state (`opened`, `closed`) and labels
- **GitLab API pagination** — handles projects with many issues
- **Cross-platform** — compiles to a single static binary for Linux, macOS, and Windows

## Installation

### From source

```bash
go install github.com/AlexFabre/gitex@latest
```

### Build locally

```bash
git clone https://github.com/AlexFabre/gitex.git
cd gitex
make build        # builds for your platform → build/gitex
make all          # cross-compiles for Linux, macOS, Windows
```

## Usage

### Configuration

The tool can be configured via CLI flags or environment variables:

| Flag           | Env var          | Description                          | Default             |
|----------------|------------------|--------------------------------------|---------------------|
| `--gitlab-url` | `GITLAB_URL`     | GitLab instance URL                  | `https://gitlab.com`|
| `--token`      | `GITLAB_TOKEN`   | GitLab private token **(required)**  |                     |
| `--project`    | `GITLAB_PROJECT` | Project path **(required)**          |                     |
| `--output`     | `GITLAB_OUTPUT`  | Output directory                     | `./output`          |

### Export all issues

```bash
gitex issues \
  --gitlab-url https://gitlab.example.com \
  --project my-group/my-project
```

### Export a single issue

```bash
gitex issues --issue-id 42 \
  --gitlab-url https://gitlab.example.com \
  --project my-group/my-project
```

### Filter by state and labels

```bash
gitex issues \
  --state opened \
  --labels "bug,critical" \
  --gitlab-url https://gitlab.example.com \
  --project my-group/my-project
```

## Output structure

```txt
output/
├── issues/
│   ├── issue-1.md
│   ├── issue-2.md
│   └── images/
│       ├── issue-1-image-1.png
│       ├── issue-2-image-1.jpg
│       └── issue-2-image-2.png
└── appendix/
    ├── issue-3-schematic.pdf
    ├── issue-5-data.xlsx
    └── issue-7-architecture.drawio
```

## Draw.io support

If the `drawio` CLI is available on your system, `.drawio` attachments are automatically rendered to PNG and embedded as images in the markdown, with a link to the source file below.

The tool looks for `drawio` or `draw.io` on PATH

## Generated markdown format

```markdown
# #42 — Issue Title

- **State**: opened
- **Author**: john.doe
- **Created**: 2025-01-15
- **Labels**: bug, critical
- **Assignees**: jane.smith
- **Milestone**: v2.0
- **URL**: https://gitlab.example.com/group/project/-/issues/42

---

## Description

Issue body with local image paths...

---

## Comments

### john.doe — 2025-01-16 10:30

Comment body...
```
