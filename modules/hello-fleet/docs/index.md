# Hello Fleet

A reference module. It ships a declarative page, a widget and a job type, and uses only the host
functions core provides: nothing here reaches the filesystem, the network or the clock directly.

- **Page** shows what the module was told about its invocation: which site it is running on, and the
  time according to the host.
- **Job** (`say-hello`) greets each target. Run it as a dry run first; it changes nothing either way.
