# HomeBoard — Developer Friction Log

This log captures real engineering blockers, environment quirks, and technical hurdles encountered during the development of HomeBoard for Amazon Fire TV / Vega OS and AWS.

| Date | Component | Issue / Friction | Root Cause | Resolution / Workaround | Status |
|---|---|---|---|---|---|
| 2026-10-03 | Repository Setup | Initial repository initialization and security gating | Confidential project specifications must not be leaked into public VCS | Added explicit `.gitignore` rules for hackathon instruction files, configured zero-secret enforcement | Resolved |
