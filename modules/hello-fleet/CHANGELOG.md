# Changelog

## 0.2.0

Uses the host functions core added in Phase 4. The page now reads the site it is running on back
through `sites.current()` rather than trusting what it was handed, shows that site's time zone, and
counts how often it has been opened in this module's own storage, which is per site and survives the
invocation. A page action asks core to run the module's own `say-hello` job and reports what came
back, including a refusal.

Nothing new is asked for: the permissions, connection privileges and network hosts are unchanged
from 0.1.0, so this update applies without needing review.

## 0.1.0

First release. Declarative page, widget and a `say-hello` job type.
