#!/usr/bin/env bash
set +e

"${HOME}/.local/bin/bbctl" login --no-desktop
status=$?

echo
if [[ ${status} -eq 0 ]]; then
    echo "Beeper bbctl login completed successfully."
else
    echo "Beeper bbctl login failed with exit code ${status}."
fi
echo "This window will remain open so you can read the result."
read -r -p "Press Enter to close... "
exit "${status}"
