import assert from 'node:assert/strict';
import { test } from 'node:test';
import { buildDbTemplateSqlCopyText, buildDbTemplateSqlSection } from './dbTemplateCopy.ts';

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
