#!/usr/bin/env python3
"""Refactor dashboard page files to use shared dashboard_shell template.

For each file:
1. Remove shell boilerplate (DOCTYPE through </div> closing flex container)
2. Remove trailing shared components (quick_order_modal, etc.) + </body></html>
3. Insert {{template "dashboard_shell" .}} at top
4. Rename content template names to dashboard/ prefix where needed
5. Move any <style> from head into content define
"""

import re
import os
import shutil

BASE = os.path.dirname(os.path.abspath(__file__))

def read_file(path):
    with open(path, 'r', encoding='utf-8') as f:
        return f.read()

def write_file(path, content):
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)
    print(f"  Wrote {path}")

def find_lines(content, pattern):
    """Find all line numbers (0-indexed) matching pattern."""
    lines = content.split('\n')
    return [i for i, line in enumerate(lines) if pattern in line]

# === File configurations ===
# Each entry: (filepath, template_name, has_extra_defines, extra_head_style, trailing_rm_start_marker, no_trailing, is_reports_style, is_dashboard_style)

files_config = [
    # (relative_path, content_template_name, has_extra_defines, extra_head_style,
    #  trailing_rm_marker, no_trailing_boilerplate, is_reports_style, is_dashboard_style)

    ("web/templates/pages/business/dashboard/products.html",
     "dashboard/products_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/orders.html",
     "dashboard/orders_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/bookings.html",
     "dashboard/bookings_content", False, False,
     '{{template "assist_panel" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/services.html",
     "dashboard/services_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/analytics.html",
     "dashboard/analytics_content", True, '  .bar { height: 8px;',
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/payments.html",
     "dashboard/payments_content", True, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/reviews.html",
     "dashboard/reviews_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/hours.html",
     "dashboard/hours_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/locations.html",
     "dashboard/locations_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard/team.html",
     "dashboard/team_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/dashboard.html",
     "dashboard_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, True),

    ("web/templates/pages/business/dashboard/reports.html",
     "dashboard/reports_content", False, False,
     '{{template "quick_order_modal" .}}', False, True, False),

    ("web/templates/pages/business/business_share.html",
     "dashboard/share_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),

    ("web/templates/pages/business/notification_settings.html",
     "dashboard/notification_settings_content", False, False,
     '{{template "quick_order_modal" .}}', False, False, False),
]

def process_file(filepath, template_name, has_extra_defines, extra_head_style,
                 trailing_marker, no_trailing, is_reports, is_dashboard):
    path = os.path.join(BASE, filepath)
    content = read_file(path)
    lines = content.split('\n')

    # === Find key landmarks ===
    # 1. Closing </div> of the flex container (end of shell)
    # 2. {{define ...}} line (start of content define)
    # 3. {{end}} that closes the content define
    # 4. Start of trailing boilerplate

    define_lines = find_lines(content, '{{define')
    end_of_shell_line = None
    start_of_trailing_line = None

    for i, line in enumerate(lines):
        stripped = line.strip()
        # Find the closing </div> of flex container
        if stripped == '</div>' and end_of_shell_line is None:
            # Check if previous non-empty line has "dashboard_sidebar"
            for j in range(i-1, -1, -1):
                if 'dashboard_sidebar' in lines[j]:
                    end_of_shell_line = i + 1  # include the blank line after
                    break
                if lines[j].strip():
                    break

    # Find the content define line
    content_define_line = None
    for dl in define_lines:
        if 'dashboard/' in lines[dl] or template_name.replace('dashboard/', '') in lines[dl]:
            content_define_line = dl
            break

    if content_define_line is None:
        # If no matching define found, use the first define
        content_define_line = define_lines[0] if define_lines else None

    # Find the {{end}} that closes the content define
    # We need to find the matching {{end}} for the content define
    content_end_line = None
    if content_define_line is not None:
        depth = 1
        for i in range(content_define_line + 1, len(lines)):
            stripped = lines[i].strip()
            if stripped.startswith('{{define') or stripped.startswith('{{block'):
                depth += 1
            elif stripped == '{{end}}' or stripped == '{{end}}':
                depth -= 1
                if depth == 0:
                    content_end_line = i
                    break

    # If we couldn't find landmarks, skip
    if end_of_shell_line is None:
        print(f"  ERROR: Could not find end of shell in {filepath}")
        return
    if content_define_line is None:
        print(f"  ERROR: Could not find content define in {filepath}")
        return
    if content_end_line is None:
        print(f"  ERROR: Could not find end of content define in {filepath}")
        return

    # === Build new content ===
    new_lines = []

    # Line 1: {{template "dashboard_shell" .}}
    new_lines.append('{{template "dashboard_shell" .}}')
    new_lines.append('')

    # For reports.html-style: no define wrapper, need to wrap content
    if is_reports:
        new_lines.append('{{define "' + template_name + '"}}')
        # Copy content between end_of_shell and trailing marker
        # Content starts at end_of_shell line (the blank line after </div>)
        # or the line that starts with the content itself
        # Trailing marker is where shared components begin
        trailing_lines = [i for i, line in enumerate(lines) if trailing_marker in line]
        if trailing_lines:
            content_start = end_of_shell_line
            content_end_trailing = trailing_lines[0]
            for i in range(content_start, content_end_trailing):
                new_lines.append(lines[i])
        new_lines.append('{{end}}')
        new_lines.append('')
    elif is_dashboard:
        # dashboard.html: no define, uses dashboard_content from components
        # Just add the shell call, that's it
        pass
    else:
        # Standard: copy the define block (including its own {{define}} and {{end}})
        for i in range(content_define_line, content_end_line + 1):
            new_lines.append(lines[i])
        new_lines.append('')

    # If there are extra defines (like analytics_stats_grid, payments_stats_grid)
    if has_extra_defines:
        # Copy everything after content_end_line up to trailing marker
        trailing_lines = [i for i, line in enumerate(lines) if trailing_marker in line]
        if trailing_lines:
            trailing_start = trailing_lines[0]
            for i in range(content_end_line + 1, trailing_start):
                new_lines.append(lines[i])
            new_lines.append('')

    # If no trailing boilerplate and we need extra defines
    if no_trailing and has_extra_defines:
        for i in range(content_end_line + 1, len(lines)):
            new_lines.append(lines[i])

    # Remove trailing empty lines
    while new_lines and new_lines[-1] == '':
        new_lines.pop()

    result = '\n'.join(new_lines) + '\n'
    write_file(path, result)
    print(f"  Refactored {filepath}: {template_name}")


# Process each file
for config in files_config:
    process_file(*config)

print("\nDone processing files.")
