#!/bin/bash

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
AUTH_USER="${AUTH_USER:-admin}"
AUTH_PASS="${AUTH_PASS:-secret}"

echo "=========================================="
echo "Prosig API Test Script"
echo "=========================================="
echo "Base URL: $BASE_URL"
echo "Auth: $AUTH_USER:$AUTH_PASS"
echo ""

check_server() {
    echo "Checking if server is running..."
    local max_attempts=5
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f --user "$AUTH_USER:$AUTH_PASS" "$BASE_URL/metrics" > /dev/null 2>&1 || \
           curl -s -f "$BASE_URL/api/posts" --user "$AUTH_USER:$AUTH_PASS" > /dev/null 2>&1; then
            echo "✓ Server is running"
            echo ""
            return 0
        fi
        echo "  Attempt $attempt/$max_attempts: Waiting for server..."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    echo "ERROR: Server is not responding at $BASE_URL after $max_attempts attempts"
    echo "Please start the server first: go run ."
    exit 1
}

test_create_post() {
    echo "1. Creating a new blog post..."
    RESPONSE=$(http --ignore-stdin --json POST "$BASE_URL/api/posts" \
        title="My First Blog Post" \
        content="This is the content of my first blog post. It's really exciting!" \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=HhBb 2>&1)
    
    if echo "$RESPONSE" | grep -q "201\|200"; then
        echo "✓ Post created successfully"
        POST_ID=$(echo "$RESPONSE" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
        echo "  Post ID: $POST_ID"
    else
        echo "✗ Failed to create post"
        echo "$RESPONSE"
    fi
    echo ""
}

test_list_posts() {
    echo "2. Listing all blog posts..."
    http --ignore-stdin GET "$BASE_URL/api/posts" \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=HhBb
    echo ""
}

test_get_post() {
    local post_id="${1:-1}"
    echo "3. Getting blog post ID: $post_id..."
    http --ignore-stdin GET "$BASE_URL/api/posts/$post_id" \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=HhBb
    echo ""
}

test_create_comment() {
    local post_id="${1:-1}"
    echo "4. Adding comment to post ID: $post_id..."
    http --ignore-stdin --json POST "$BASE_URL/api/posts/$post_id/comments" \
        content="This is a great post! Thanks for sharing." \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=HhBb
    echo ""
}

test_create_second_comment() {
    local post_id="${1:-1}"
    echo "5. Adding second comment to post ID: $post_id..."
    http --ignore-stdin --json POST "$BASE_URL/api/posts/$post_id/comments" \
        content="I completely agree with your points." \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=HhBb
    echo ""
}

test_create_second_post() {
    echo "6. Creating a second blog post..."
    http --ignore-stdin --json POST "$BASE_URL/api/posts" \
        title="Second Post" \
        content="This is my second blog post with more interesting content." \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=HhBb
    echo ""
}

test_list_posts_with_counts() {
    echo "7. Listing all posts (should show comment counts)..."
    http --ignore-stdin GET "$BASE_URL/api/posts" \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=HhBb
    echo ""
}

test_get_post_with_comments() {
    local post_id="${1:-1}"
    echo "8. Getting post ID: $post_id (with comments)..."
    http --ignore-stdin GET "$BASE_URL/api/posts/$post_id" \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=HhBb
    echo ""
}

test_health() {
    echo "9. Health check..."
    http --ignore-stdin GET "$BASE_URL/health" \
        --print=HhBb
    echo ""
}

test_metrics() {
    echo "10. Getting metrics..."
    http --ignore-stdin GET "$BASE_URL/metrics" \
        --print=HhBb | head -40
    echo ""
}

test_session_auth() {
    echo "10. Testing session-based authentication..."
    echo "    First request with Basic Auth (creates session)..."
    SESSION_COOKIE=$(http --ignore-stdin GET "$BASE_URL/api/posts" \
        --auth "$AUTH_USER:$AUTH_PASS" \
        --print=Hh 2>&1 | grep -i "set-cookie" | head -1 | sed 's/.*session_id=\([^;]*\).*/\1/')
    
    if [ -n "$SESSION_COOKIE" ]; then
        echo "    ✓ Session cookie received: ${SESSION_COOKIE:0:20}..."
        echo "    Making request with session cookie..."
        http --ignore-stdin GET "$BASE_URL/api/posts" \
            Cookie:"session_id=$SESSION_COOKIE" \
            --print=HhBb
    else
        echo "    ✗ Failed to get session cookie"
    fi
    echo ""
}

main() {
    check_server
    
    test_create_post
    test_list_posts
    test_get_post 1
    test_create_comment 1
    test_create_second_comment 1
    test_get_post 1
    test_create_second_post
    test_list_posts_with_counts
    test_get_post 2
    test_health
    test_metrics
    test_session_auth
    
    echo "=========================================="
    echo "All tests completed!"
    echo "=========================================="
}

main "$@"
