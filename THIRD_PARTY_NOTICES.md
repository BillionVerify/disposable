# Third-Party Notices

This repository is MIT-licensed. The embedded disposable-domain data
(`data/domains.txt`) is maintained by BillionVerify and may include entries
merged from the following community projects. Each upstream's license is
verified on every hourly refresh (`scripts/sources.json` declares the
expected license; `scripts/merge_upstream.py` checks that the upstream's
license file still matches before pulling data).

## disposable-email-domains/disposable-email-domains

- Source: https://github.com/disposable-email-domains/disposable-email-domains
- License: CC0-1.0 (Public Domain Dedication)

## FGRibreau/mailchecker

- Source: https://github.com/FGRibreau/mailchecker
- License: MIT — Copyright (c) 2013 Francois-Guillaume Ribreau

## 7c/fakefilter

- Source: https://github.com/7c/fakefilter
- License: BSD-3-Clause — Copyright (c) 2022, 7c

## amieiro/disposable-email-domains

- Source: https://github.com/amieiro/disposable-email-domains
- License: MIT

## ivolo/disposable-email-domains

- Source: https://github.com/ivolo/disposable-email-domains
- License: MIT

## wesbos/burner-email-providers

- Source: https://github.com/wesbos/burner-email-providers
- License: MIT — Copyright 2018 Wes Bos

## martenson/disposable-email-domains

- Source: https://github.com/martenson/disposable-email-domains
- License: CC0-1.0 (Public Domain Dedication)

## groundcat/disposable-email-domain-list

- Source: https://github.com/groundcat/disposable-email-domain-list
- License: MIT

## unkn0w/disposable-email-domain-list

- Source: https://github.com/unkn0w/disposable-email-domain-list
- License: MIT — Copyright (c) 2022 Jakub 'unknow' Mrugalski

---

All upstream entries are normalized, deduplicated, and subject to
`data/exceptions.txt` (our never-flag list) before they appear in
`data/domains.txt`. Update this notice — and `scripts/sources.json` —
before adding any new data source.
