import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  buildDbTemplateSqlCopyText,
  buildDbTemplateSqlSection,
  mergeDbTemplatePlaceholders,
  parseDbTemplatePlaceholders,
} from './dbTemplateCopy.ts';

test('appends the type prompt below the SQL when copying a database template', () => {
  const section = buildDbTemplateSqlSection(
    { ID: 127, projectName: '示例项目' },
    {
      ID: 2,
      typeName: '建表',
      prompt: '根据产品需求改造建表 SQL。\n只输出 SQL。',
    },
    {
      scriptName: '建表 SQL',
      content: 'CREATE TABLE demo (id int8);',
    },
  );

  assert.equal(
    section,
    [
      '-- 项目：示例项目',
      '-- 业务类型：建表',
      '-- 脚本：建表 SQL',
      '',
      'CREATE TABLE demo (id int8);',
      '',
      '-- 提示词：',
      '-- 根据产品需求改造建表 SQL。',
      '-- 只输出 SQL。',
    ].join('\n'),
  );
});

test('omits the prompt block when the business type prompt is empty', () => {
  const section = buildDbTemplateSqlSection(
    { ID: 127, projectName: '示例项目' },
    { ID: 3, typeName: '字典', prompt: '   ' },
    {
      scriptName: '字典 SQL',
      content: 'INSERT INTO sys_dict_data VALUES (...);',
    },
  );

  assert.equal(
    section,
    [
      '-- 项目：示例项目',
      '-- 业务类型：字典',
      '-- 脚本：字典 SQL',
      '',
      'INSERT INTO sys_dict_data VALUES (...);',
    ].join('\n'),
  );
});

test('joins all SQL sections with wide spacing for copy preview', () => {
  assert.equal(
    buildDbTemplateSqlCopyText(['section A', 'section B']),
    'section A\n\n\nsection B',
  );
});

test('applies dynamic placeholder values when copying a database template', () => {
  const section = buildDbTemplateSqlSection(
    { ID: 127, projectName: '示例项目' },
    {
      typeName: '公司菜单',
      prompt: 'menu_code 使用 {{menuCode}}。',
    },
    {
      scriptName: '公司菜单',
      content: "INSERT INTO sys_company_menu(menu_code, company_id) VALUES ('{{menuCode}}', ${companyId});",
    },
    {
      menuCode: 'mtp.menu.demo',
      companyId: '1001',
    },
  );

  assert.equal(section.includes("'mtp.menu.demo', 1001"), true);
  assert.equal(section.includes('-- menu_code 使用 mtp.menu.demo。'), true);
});

test('supports placeholder keys entered with delimiters', () => {
  const section = buildDbTemplateSqlSection(
    { ID: 127, projectName: '示例项目' },
    { typeName: '菜单', prompt: '' },
    {
      scriptName: '菜单 SQL',
      content: 'VALUES (\n    {{menu_parent_id}},\n    ${menu_code}\n);',
    },
    {
      '{{menu_parent_id}}': '2063000000000000004',
      '${menu_code}': "'mtp.menu.demo'",
    },
  );

  assert.equal(section.includes('{{menu_parent_id}}'), false);
  assert.equal(section.includes('${menu_code}'), false);
  assert.equal(section.includes('2063000000000000004'), true);
  assert.equal(section.includes("'mtp.menu.demo'"), true);
});

test('parses and merges dynamic placeholders by key', () => {
  assert.deepEqual(
    parseDbTemplatePlaceholders('[{"key":"{{companyId}}","description":"公司","value":"-1"}]'),
    [{ key: 'companyId', description: '公司', value: '-1' }],
  );

  assert.deepEqual(
    mergeDbTemplatePlaceholders([
      { dynamicPlaceholders: '[{"key":"companyId","description":"公司","value":"-1"}]' },
      { dynamicPlaceholders: '[{"key":"companyId","description":"重复","value":"100"}]' },
      { dynamicPlaceholders: '[{"key":"menuCode","description":"菜单","value":""}]' },
    ]),
    [
      { key: 'companyId', description: '公司', value: '-1' },
      { key: 'menuCode', description: '菜单', value: '' },
    ],
  );
});
