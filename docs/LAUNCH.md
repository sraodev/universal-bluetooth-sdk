# Repository reach and launch checklist

Research and copy prepared 2026-08-30. The aim is better comprehension, easier
first use, and useful contributions. No star, ranking, or CTR outcome is promised.

## GitHub About copy

> Experimental Go Bluetooth daemon and CLI with Linux RFCOMM, MCP tools, and a hardware-free demo. Building toward nearby chat and optional local AI.

About copy and the following topics were applied on 2026-08-30. Topics are limited
to capabilities or the clearly stated project domain:

`bluetooth`, `rfcomm`, `bluez`, `golang`, `raspberry-pi`, `iot`, `cli`,
`mcp-server`, `developer-tools`, `open-source`, `local-ai`

Remove obsolete discovery terms such as `pickle` and `python-script` from the main
positioning. Do not add `bitchat-compatible`, `encrypted-messenger`, or `mesh-network`
as if implemented. BLE may be described in the roadmap, not as a shipped backend.
[GitHub topics guidance](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/classifying-your-repository-with-topics).

## Preview asset

Upload [`media/social-preview.png`](../media/social-preview.png) in repository
Settings → General → Social preview. It is an original 1280×640 PNG, not a screenshot
of working chat. Source: [`generate_preview.py`](../media/generate_preview.py), using
Pillow as an optional asset-authoring dependency; the SDK does not depend on it.
The generator's font path targets macOS; change it for another authoring host.
[GitHub preview guidance](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/customizing-your-repositorys-social-media-preview).

## Before announcing

- Merge reviewed changes; verify both CI jobs on GitHub. Only then add a CI badge
  and require those exact successful check names for merging.
- Master is protected from deletion and force pushes by an active ruleset created
  2026-08-30. Required status checks are intentionally not configured yet.
- About/topics copy is applied. Social-preview upload is a separate repository
  setting; verify the live preview after uploading.
- Private vulnerability reporting was enabled on 2026-08-30. Use the repository
  Security tab for sensitive reports; no response SLA is promised.
- Record a 20–30 second terminal demonstration using `scripts/smoke.py` and the
  quick start. Label synthetic peers clearly. Add a real chat video only after
  receiving and physical-device tests exist.
- Release an experimental tag only after reviewing artifact provenance, checksums,
  license notices, install/uninstall instructions, and hardware limitations. Do
  not label the current code 1.0 or claim certified cross-platform support.

## Suggested announcement (draft; not posted)

> I'm building Universal Bluetooth SDK: a Go daemon and CLI for Bluetooth
> experiments, with an MCP interface and a demo that needs no radio or API key.
> Linux RFCOMM sending works in code; real-device validation and two-way sessions
> are the next milestones. BLE chat is on the roadmap, and Bitchat compatibility
> is not claimed. Looking for Raspberry Pi testers and Go/BLE contributors.
> Try the stub demo and choose a scoped issue:
> https://github.com/sraodev/universal-bluetooth-sdk

Share a useful technical walkthrough in relevant Go, Raspberry Pi, and Bluetooth
communities only when their rules allow it. Explain the limitations and ask for
specific test reports. Avoid mass posting, unsolicited DMs, star exchanges, or
keyword stuffing. Maintain a few approachable issues with acceptance criteria;
GitHub can surface `good first issue` work in discovery.
[GitHub contribution labels guidance](https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/encouraging-helpful-contributions-to-your-project-with-labels).

## Measure weekly, interpret cautiously

Track GitHub unique visitors/cloners, referrers, stars, demo failure reports, and
first-time contributor PRs. Compare like-for-like periods before and after one
change. A platform's post impressions and link clicks can give that post's CTR;
repository stars/views alone cannot give impression-based CTR. Do not attribute
all growth to a README change. Prioritize successful setup and repeat contributors.
