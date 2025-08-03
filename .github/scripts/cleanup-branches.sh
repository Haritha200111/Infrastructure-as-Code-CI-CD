set -e

DEFAULT_BRANCH="main"
PROTECTED_BRANCHES=("main" "master" "development")
CUTOFF_DAYS=90
NOW=$(date +%s)

OLD_BRANCHES=()
DELETED_BRANCHES=()

# Get all remote branches except HEAD
BRANCHES=$(git for-each-ref --format='%(refname:short)' refs/remotes/origin | grep -vE "HEAD|origin/$DEFAULT_BRANCH")

for BRANCH in $BRANCHES; do
  SHORT_NAME=${BRANCH#origin/}

  # Skip protected
  if [[ " ${PROTECTED_BRANCHES[@]} " =~ " ${SHORT_NAME} " ]]; then
    continue
  fi

  # Get last commit date
  LAST_COMMIT_DATE=$(git log -1 --format=%ct "$BRANCH" || echo 0)
  AGE_DAYS=$(( (NOW - LAST_COMMIT_DATE) / 86400 ))

  if [ "$AGE_DAYS" -gt "$CUTOFF_DAYS" ]; then
    OLD_BRANCHES+=("$SHORT_NAME (last updated ${AGE_DAYS} days ago)")
  fi

  # Check if merged into main
  git fetch origin "$DEFAULT_BRANCH" > /dev/null 2>&1
  if git branch -r --merged "origin/$DEFAULT_BRANCH" | grep -q "origin/$SHORT_NAME"; then
    git push origin --delete "$SHORT_NAME" && DELETED_BRANCHES+=("$SHORT_NAME")
  fi
done

# Set outputs
echo "oldBranches<<EOF" >> $GITHUB_OUTPUT
printf "%s\n" "${OLD_BRANCHES[@]}" >> $GITHUB_OUTPUT
echo "EOF" >> $GITHUB_OUTPUT

echo "deletedBranches<<EOF" >> $GITHUB_OUTPUT
printf "%s\n" "${DELETED_BRANCHES[@]}" >> $GITHUB_OUTPUT
echo "EOF" >> $GITHUB_OUTPUT
