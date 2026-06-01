#!/usr/bin/env python3
"""Patch InterfaceManager.tsx: add lastTestedAt column and data cell."""

import os

FILE = os.path.join(os.path.dirname(__file__), '..', 'web-react', 'src', 'views', 'interface-manager', 'InterfaceManager.tsx')

with open(FILE, 'r', encoding='utf-8') as f:
    lines = f.readlines()

new_lines = []
i = 0
patched_header = False
patched_td = False

while i < len(lines):
    line = lines[i]
    
    # 1. Insert "最近测试" th before "操作" th (in the table header)
    if not patched_header and '操作</th>' in line and 'font-bold text-center' in line:
        # Get leading whitespace from this line
        ws = line[:len(line) - len(line.lstrip())]
        new_lines.append(f'{ws}<th className="px-6 py-4 font-bold">最近测试</th>\n')
        new_lines.append(line)
        patched_header = True
        i += 1
        continue
    
    # 2. Insert lastTestedAt td before the 操作 td (in the table body)
    if not patched_td and 'text-center space-x-3 whitespace-nowrap' in line and '<td' in line:
        ws = line[:len(line) - len(line.lstrip())]
        inner_ws = ws + '  '
        new_lines.append(f'{ws}<td className="px-6 py-4 whitespace-nowrap">\n')
        new_lines.append(f'{inner_ws}{{iface.lastTestedAt ? (\n')
        new_lines.append(f'{inner_ws}  <span className="inline-flex items-center gap-1.5 text-xs">\n')
        new_lines.append(f'{inner_ws}    <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0"></span>\n')
        new_lines.append(f'{inner_ws}    <span className="text-slate-600" title={{new Date(iface.lastTestedAt).toLocaleString()}}>{{formatRelativeTime(iface.lastTestedAt)}}</span>\n')
        new_lines.append(f'{inner_ws}  </span>\n')
        new_lines.append(f'{inner_ws}) : (\n')
        new_lines.append(f'{inner_ws}  <span className="text-xs text-slate-300">未测试</span>\n')
        new_lines.append(f'{inner_ws})}}\n')
        new_lines.append(f'{ws}</td>\n')
        new_lines.append(line)  # keep the original 操作 td
        patched_td = True
        i += 1
        continue
    
    new_lines.append(line)
    i += 1

with open(FILE, 'w', encoding='utf-8') as f:
    f.writelines(new_lines)

print(f"✅ Done! patched_header={patched_header}, patched_td={patched_td}")
