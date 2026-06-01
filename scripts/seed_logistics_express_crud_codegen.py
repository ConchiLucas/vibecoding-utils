#!/usr/bin/env python3
"""
Seed easy-deploy code-generation templates for the zhongtie logistics express
single-table CRUD style.

The generated easy-deploy project uses this disk path as its root:
  logistics-express-manage-starter

Files that belong to logistics-express-manage.common are generated with a
relative "../logistics-express-manage.common/..." path from that root.
"""

from __future__ import annotations

import argparse
import sys
from dataclasses import dataclass


DEFAULT_DSN = "postgresql://conchi:conchi123456@127.0.0.1:5432/easy_deploy"
DEFAULT_PROJECT_NAME = "zhwl-logistics-express-single-crud"
DEFAULT_DISK_PATH = (
    "/Users/conchi/workforce/company_workforce/zhongtie_workforce/"
    "zhwl-logistics-express/logistics-express-manage/"
    "logistics-express-manage-starter"
)

STARTER_JAVA = "src/main/java/com/wl/logistics/express/manage/starter"
COMMON_JAVA = (
    "../logistics-express-manage.common/src/main/java/"
    "com/wl/logistics/express/manage/common"
)


def java_type_action() -> str:
    return '{[< if eq .JavaDataType "Date" >]}LocalDateTime{[< else >]}{[< .JavaDataType >]}{[< end >]}'


def skip_base_start() -> str:
    return (
        '{[< if and (ne .OriginalName "id") (ne .OriginalName "is_del") '
        '(ne .OriginalName "create_time") (ne .OriginalName "creator") '
        '(ne .OriginalName "update_time") (ne .OriginalName "modifier") '
        '(ne .OriginalName "tenant_id") >]}'
    )


SKIP_BASE_END = "{[< end >]}"


@dataclass(frozen=True)
class TemplateNode:
    file_url: str
    file_name: str
    content: str
    enabled: int = 1
    incremented: int = 0


CONTROLLER_TEMPLATE = """package {[< .starterPackage >]}.controller;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.wl.logistics.common.vo.PageQueryVo;
import com.wl.logistics.common.vo.R;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}AddDTO;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}EditDTO;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}PageDTO;
import {[< .commonPackage >]}.entity.{[< .ModuleName >]};
import {[< .commonPackage >]}.vo.{[< .ModuleName >]}DetailVO;
import {[< .commonPackage >]}.vo.{[< .ModuleName >]}PageVO;
import {[< .starterPackage >]}.service.{[< .ModuleName >]}Service;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import javax.annotation.Resource;

/**
 * {[< .moduleComment >]} controller.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@RestController
@RequestMapping("/{[< .moduleName >]}")
public class {[< .ModuleName >]}Controller {

    @Resource
    private {[< .ModuleName >]}Service {[< .moduleName >]}Service;

    @PostMapping("/queryPage")
    public R<Page<{[< .ModuleName >]}PageVO>> queryPage(@RequestBody PageQueryVo<{[< .ModuleName >]}PageDTO> pageQuery) {
        Page<{[< .ModuleName >]}PageVO> pageVO = {[< .moduleName >]}Service.queryPage(pageQuery);
        return R.ok(pageVO);
    }

    @GetMapping("/detail")
    public R<{[< .ModuleName >]}DetailVO> detail(String id) {
        return R.ok({[< .moduleName >]}Service.detail(id));
    }

    @PostMapping("/add")
    public R<String> add(@RequestBody {[< .ModuleName >]}AddDTO addDTO) {
        boolean result = {[< .moduleName >]}Service.add(addDTO);
        return result ? R.ok(null, "新增成功") : R.fail("新增失败");
    }

    @PostMapping("/edit")
    public R<String> edit(@RequestBody {[< .ModuleName >]}EditDTO editDTO) {
        boolean result = {[< .moduleName >]}Service.edit(editDTO);
        return result ? R.ok("编辑成功") : R.fail("编辑失败");
    }

    @PostMapping("/delete")
    public R<String> delete(@RequestBody {[< .ModuleName >]} entity) {
        boolean result = {[< .moduleName >]}Service.delete(entity.getId());
        return result ? R.ok("删除成功") : R.fail("删除失败");
    }
}
"""


SERVICE_TEMPLATE = """package {[< .starterPackage >]}.service;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.baomidou.mybatisplus.extension.service.IService;
import com.wl.logistics.common.vo.PageQueryVo;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}AddDTO;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}EditDTO;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}PageDTO;
import {[< .commonPackage >]}.entity.{[< .ModuleName >]};
import {[< .commonPackage >]}.vo.{[< .ModuleName >]}DetailVO;
import {[< .commonPackage >]}.vo.{[< .ModuleName >]}PageVO;

/**
 * {[< .moduleComment >]} service.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
public interface {[< .ModuleName >]}Service extends IService<{[< .ModuleName >]}> {

    Page<{[< .ModuleName >]}PageVO> queryPage(PageQueryVo<{[< .ModuleName >]}PageDTO> pageQuery);

    {[< .ModuleName >]}DetailVO detail(String id);

    boolean add({[< .ModuleName >]}AddDTO addDTO);

    boolean edit({[< .ModuleName >]}EditDTO editDTO);

    boolean delete(String id);
}
"""


SERVICE_IMPL_TEMPLATE = """package {[< .starterPackage >]}.service.impl;

import cn.hutool.core.bean.BeanUtil;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.wl.logistics.common.starter.utils.PageableUtil;
import com.wl.logistics.common.vo.PageQueryVo;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}AddDTO;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}EditDTO;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}PageDTO;
import {[< .commonPackage >]}.entity.{[< .ModuleName >]};
import {[< .commonPackage >]}.vo.{[< .ModuleName >]}DetailVO;
import {[< .commonPackage >]}.vo.{[< .ModuleName >]}PageVO;
import {[< .starterPackage >]}.mapper.{[< .ModuleName >]}Mapper;
import {[< .starterPackage >]}.service.{[< .ModuleName >]}Service;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * {[< .moduleComment >]} service implementation.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@Service
public class {[< .ModuleName >]}ServiceImpl extends ServiceImpl<{[< .ModuleName >]}Mapper, {[< .ModuleName >]}> implements {[< .ModuleName >]}Service {

    @Override
    public Page<{[< .ModuleName >]}PageVO> queryPage(PageQueryVo<{[< .ModuleName >]}PageDTO> pageQuery) {
        Page<?> page = PageableUtil.buildPageQuery(pageQuery);
        {[< .ModuleName >]}PageDTO queryParams = pageQuery.getParams();
        return baseMapper.select{[< .ModuleName >]}Page(page, queryParams);
    }

    @Override
    public {[< .ModuleName >]}DetailVO detail(String id) {
        {[< .ModuleName >]} entity = this.getById(id);
        if (entity == null) {
            return null;
        }
        {[< .ModuleName >]}DetailVO vo = new {[< .ModuleName >]}DetailVO();
        BeanUtil.copyProperties(entity, vo);
        return vo;
    }

    @Override
    @Transactional(rollbackFor = Exception.class)
    public boolean add({[< .ModuleName >]}AddDTO addDTO) {
        {[< .ModuleName >]} entity = new {[< .ModuleName >]}();
        BeanUtil.copyProperties(addDTO, entity);
        entity.setIsDel("0");
        return this.save(entity);
    }

    @Override
    @Transactional(rollbackFor = Exception.class)
    public boolean edit({[< .ModuleName >]}EditDTO editDTO) {
        {[< .ModuleName >]} entity = new {[< .ModuleName >]}();
        BeanUtil.copyProperties(editDTO, entity);
        return this.updateById(entity);
    }

    @Override
    @Transactional(rollbackFor = Exception.class)
    public boolean delete(String id) {
        return this.removeById(id);
    }
}
"""


MAPPER_TEMPLATE = """package {[< .starterPackage >]}.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import {[< .commonPackage >]}.dto.{[< .ModuleName >]}PageDTO;
import {[< .commonPackage >]}.entity.{[< .ModuleName >]};
import {[< .commonPackage >]}.vo.{[< .ModuleName >]}PageVO;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

/**
 * {[< .moduleComment >]} mapper.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@Mapper
public interface {[< .ModuleName >]}Mapper extends BaseMapper<{[< .ModuleName >]}> {

    Page<{[< .ModuleName >]}PageVO> select{[< .ModuleName >]}Page(Page<?> page, @Param("query") {[< .ModuleName >]}PageDTO query);
}
"""


MAPPER_XML_TEMPLATE = """<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="{[< .starterPackage >]}.mapper.{[< .ModuleName >]}Mapper">

    <sql id="{[< .moduleName >]}QueryCondition">
        t.is_del = '0'
        <if test="query != null">
{[< range .field_lines >]}""" + skip_base_start() + """
            <if test="query.{[< .JavaField >]} != null{[< if eq .JavaDataType "String" >]} and query.{[< .JavaField >]} != ''{[< end >]}">
                AND t.{[< .OriginalName >]} {[< if eq .JavaDataType "String" >]}LIKE '%' || #{query.{[< .JavaField >]}} || '%'{[< else >]}= #{query.{[< .JavaField >]}}{[< end >]}
            </if>
""" + SKIP_BASE_END + """{[< end >]}        </if>
    </sql>

    <select id="select{[< .ModuleName >]}Page" resultType="{[< .commonPackage >]}.vo.{[< .ModuleName >]}PageVO">
        SELECT
            t.id AS id{[< range .field_lines >]}""" + skip_base_start() + """,
            t.{[< .OriginalName >]} AS {[< .JavaField >]}""" + SKIP_BASE_END + """{[< end >]},
            t.create_time AS createTime
        FROM {[< .rawTableName >]} t
        WHERE
        <include refid="{[< .moduleName >]}QueryCondition"/>
        ORDER BY t.create_time DESC
    </select>

</mapper>
"""


ENTITY_TEMPLATE = """package {[< .commonPackage >]}.entity;

import com.baomidou.mybatisplus.annotation.TableName;
import com.wl.logistics.common.db.BaseEntity;
import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import lombok.Data;
import lombok.EqualsAndHashCode;
import lombok.experimental.Accessors;

import java.math.BigDecimal;
import java.time.LocalDateTime;

/**
 * {[< .moduleComment >]} entity.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@Data
@EqualsAndHashCode(callSuper = true)
@Accessors(chain = true)
@ApiModel(value = "{[< .moduleComment >]}", description = "{[< .moduleComment >]}")
@TableName("{[< .rawTableName >]}")
public class {[< .ModuleName >]} extends BaseEntity<{[< .ModuleName >]}> {
{[< range .field_lines >]}""" + skip_base_start() + """

    @ApiModelProperty(value = "{[< .Comment >]}")
    private """ + java_type_action() + """ {[< .JavaField >]};
""" + SKIP_BASE_END + """{[< end >]}}
"""


def dto_fields(include_id: bool) -> str:
    prefix = ""
    if include_id:
        prefix = """

    @ApiModelProperty(value = "ID", required = true)
    private String id;
"""
    return prefix + """{[< range .field_lines >]}""" + skip_base_start() + """

    @ApiModelProperty(value = "{[< .Comment >]}")
    private """ + java_type_action() + """ {[< .JavaField >]};
""" + SKIP_BASE_END + """{[< end >]}"""


ADD_DTO_TEMPLATE = """package {[< .commonPackage >]}.dto;

import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import lombok.Data;

import java.math.BigDecimal;
import java.time.LocalDateTime;

/**
 * {[< .moduleComment >]} add request.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@Data
@ApiModel(value = "{[< .moduleComment >]} add request")
public class {[< .ModuleName >]}AddDTO {""" + dto_fields(False) + """
}
"""


EDIT_DTO_TEMPLATE = """package {[< .commonPackage >]}.dto;

import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import lombok.Data;

import java.math.BigDecimal;
import java.time.LocalDateTime;

/**
 * {[< .moduleComment >]} edit request.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@Data
@ApiModel(value = "{[< .moduleComment >]} edit request")
public class {[< .ModuleName >]}EditDTO {""" + dto_fields(True) + """
}
"""


PAGE_DTO_TEMPLATE = """package {[< .commonPackage >]}.dto;

import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import lombok.Data;

import java.math.BigDecimal;
import java.time.LocalDateTime;

/**
 * {[< .moduleComment >]} page query request.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@Data
@ApiModel(value = "{[< .moduleComment >]} page query request")
public class {[< .ModuleName >]}PageDTO {""" + dto_fields(False) + """
}
"""


VO_FIELDS = """

    @ApiModelProperty(value = "ID")
    private String id;
{[< range .field_lines >]}""" + skip_base_start() + """

    @ApiModelProperty(value = "{[< .Comment >]}")
    private """ + java_type_action() + """ {[< .JavaField >]};
""" + SKIP_BASE_END + """{[< end >]}

    @ApiModelProperty(value = "create time")
    private LocalDateTime createTime;
"""


PAGE_VO_TEMPLATE = """package {[< .commonPackage >]}.vo;

import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import lombok.Data;

import java.math.BigDecimal;
import java.time.LocalDateTime;

/**
 * {[< .moduleComment >]} page response.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@Data
@ApiModel(value = "{[< .moduleComment >]} page response")
public class {[< .ModuleName >]}PageVO {""" + VO_FIELDS + """
}
"""


DETAIL_VO_TEMPLATE = """package {[< .commonPackage >]}.vo;

import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import lombok.Data;

import java.math.BigDecimal;
import java.time.LocalDateTime;

/**
 * {[< .moduleComment >]} detail response.
 *
 * @author {[< .author >]}
 * @date {[< .date >]}
 */
@Data
@ApiModel(value = "{[< .moduleComment >]} detail response")
public class {[< .ModuleName >]}DetailVO {""" + VO_FIELDS + """
}
"""


TEMPLATES = [
    TemplateNode(f"{STARTER_JAVA}/controller/", "{[<ModuleName>]}Controller.java", CONTROLLER_TEMPLATE),
    TemplateNode(f"{STARTER_JAVA}/service/", "{[<ModuleName>]}Service.java", SERVICE_TEMPLATE),
    TemplateNode(f"{STARTER_JAVA}/service/impl/", "{[<ModuleName>]}ServiceImpl.java", SERVICE_IMPL_TEMPLATE),
    TemplateNode(f"{STARTER_JAVA}/mapper/", "{[<ModuleName>]}Mapper.java", MAPPER_TEMPLATE),
    TemplateNode(f"{STARTER_JAVA}/mapper/", "{[<ModuleName>]}Mapper.xml", MAPPER_XML_TEMPLATE),
    TemplateNode(f"{COMMON_JAVA}/entity/", "{[<ModuleName>]}.java", ENTITY_TEMPLATE),
    TemplateNode(f"{COMMON_JAVA}/dto/", "{[<ModuleName>]}AddDTO.java", ADD_DTO_TEMPLATE),
    TemplateNode(f"{COMMON_JAVA}/dto/", "{[<ModuleName>]}EditDTO.java", EDIT_DTO_TEMPLATE),
    TemplateNode(f"{COMMON_JAVA}/dto/", "{[<ModuleName>]}PageDTO.java", PAGE_DTO_TEMPLATE),
    TemplateNode(f"{COMMON_JAVA}/vo/", "{[<ModuleName>]}PageVO.java", PAGE_VO_TEMPLATE),
    TemplateNode(f"{COMMON_JAVA}/vo/", "{[<ModuleName>]}DetailVO.java", DETAIL_VO_TEMPLATE),
]

PROJECT_HOLDERS = [
    ("{[<starterPackage>]}", "com.wl.logistics.express.manage.starter", "starter Java package"),
    ("{[<commonPackage>]}", "com.wl.logistics.express.manage.common", "common Java package"),
    ("{[<author>]}", "system", "generated author"),
]


def validate_templates() -> None:
    for node in TEMPLATES:
        if "{%" in node.content or "%}" in node.content:
            raise ValueError(f"Jinja syntax found in {node.file_name}")
        if "{[<" not in node.content:
            raise ValueError(f"No template placeholders found in {node.file_name}")


def connect(dsn: str):
    try:
        import psycopg2
    except ImportError as exc:
        raise SystemExit("psycopg2 is required to seed the database") from exc
    return psycopg2.connect(dsn)


def upsert_project(cur, name: str, disk_path: str, user_name: str, project_config_id: int) -> int:
    cur.execute("select id from tb_generate_project where project_name = %s", (name,))
    row = cur.fetchone()
    if row:
        project_id = row[0]
        cur.execute(
            """
            update tb_generate_project
               set disk_path = %s,
                   user_name = %s,
                   project_config_id = %s,
                   remark = %s
             where id = %s
            """,
            (disk_path, user_name, project_config_id, "zhwl logistics express single-table CRUD templates", project_id),
        )
        return project_id

    cur.execute(
        """
        insert into tb_generate_project(project_config_id, project_name, disk_path, remark, user_name)
        values (%s, %s, %s, %s, %s)
        returning id
        """,
        (project_config_id, name, disk_path, "zhwl logistics express single-table CRUD templates", user_name),
    )
    return cur.fetchone()[0]


def repair_sequences(cur) -> None:
    tables = [
        "tb_generate_project",
        "tb_generate_project_path",
        "tb_generate_project_path_model",
        "tb_generate_project_place_holder",
    ]
    for table in tables:
        cur.execute("select pg_get_serial_sequence(%s, 'id')", (table,))
        row = cur.fetchone()
        if not row or not row[0]:
            continue
        sequence_name = row[0]
        cur.execute(f"select coalesce(max(id), 0) from {table}")
        max_id = cur.fetchone()[0]
        cur.execute("select setval(%s, %s, true)", (sequence_name, max_id))


def replace_templates(cur, project_id: int) -> None:
    cur.execute("select id from tb_generate_project_path where project_id = %s", (project_id,))
    path_ids = [row[0] for row in cur.fetchall()]
    if path_ids:
        cur.execute("delete from tb_generate_project_path_model where path_id = any(%s)", (path_ids,))
    cur.execute("delete from tb_generate_project_path where project_id = %s", (project_id,))

    for node in TEMPLATES:
        cur.execute(
            """
            insert into tb_generate_project_path(project_id, file_url, file_name, enabled, incremented)
            values (%s, %s, %s, %s, %s)
            returning id
            """,
            (project_id, node.file_url, node.file_name, node.enabled, node.incremented),
        )
        path_id = cur.fetchone()[0]
        cur.execute(
            "insert into tb_generate_project_path_model(path_id, content) values (%s, %s)",
            (path_id, node.content),
        )


def replace_holders(cur, project_id: int, user_name: str) -> None:
    cur.execute("delete from tb_generate_project_place_holder where project_id = %s", (project_id,))
    for key, value, desc in PROJECT_HOLDERS:
        cur.execute(
            """
            insert into tb_generate_project_place_holder
                (project_id, user_name, holder_key, holder_value, holder_desc, example_value)
            values (%s, %s, %s, %s, %s, %s)
            """,
            (project_id, user_name, key, value, desc, value),
        )


def seed(args: argparse.Namespace) -> int:
    validate_templates()

    if args.dry_run:
        print(f"project_name={args.project_name}")
        print(f"disk_path={args.disk_path}")
        print(f"templates={len(TEMPLATES)}")
        for node in TEMPLATES:
            print(f"- {node.file_url}{node.file_name}")
        return 0

    conn = connect(args.dsn)
    try:
        with conn:
            with conn.cursor() as cur:
                repair_sequences(cur)
                project_id = upsert_project(
                    cur,
                    args.project_name,
                    args.disk_path,
                    args.user_name,
                    args.project_config_id,
                )
                replace_templates(cur, project_id)
                replace_holders(cur, project_id, args.user_name)
        print(f"Seeded project {project_id}: {args.project_name}")
        return 0
    finally:
        conn.close()


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dsn", default=DEFAULT_DSN)
    parser.add_argument("--project-name", default=DEFAULT_PROJECT_NAME)
    parser.add_argument("--disk-path", default=DEFAULT_DISK_PATH)
    parser.add_argument("--user-name", default="conchi")
    parser.add_argument("--project-config-id", type=int, default=0)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args(argv)
    return seed(args)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
