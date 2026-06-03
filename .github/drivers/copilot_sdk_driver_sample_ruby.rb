#!/usr/bin/env ruby
built_in_driver = "#{ENV.fetch('RUNNER_TEMP', '/tmp')}/gh-aw/actions/copilot_sdk_driver.cjs"
success = system("node", built_in_driver, *ARGV)
exit(success ? 0 : 1)
