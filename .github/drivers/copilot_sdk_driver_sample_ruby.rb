#!/usr/bin/env ruby
built_in_driver = "#{ENV.fetch('RUNNER_TEMP', '/tmp')}/gh-aw/actions/copilot_sdk_driver.cjs"
ok = system("node", built_in_driver, *ARGV)
exit(ok ? 0 : 1)
