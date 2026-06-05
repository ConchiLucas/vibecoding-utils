#!/usr/bin/env python3
"""
Write the real Panzhihua c12-mtp btStation CRUD code into easy-deploy project
card 127.

This script intentionally stores concrete Java/SQL files, not generation
template placeholders.
"""

from __future__ import annotations

import argparse
import sys
from dataclasses import dataclass
from pathlib import Path


DEFAULT_DSN = "postgresql://conchi:conchi123456@127.0.0.1:5432/easy_deploy"
DEFAULT_PROJECT_ID = 127
DEFAULT_PROJECT_NAME = "攀枝花-多式联运-后端"
DEFAULT_DISK_PATH = "/Users/conchi/workforce/company_workforce/panzhihua_dsly_workforce/c12-mtp"

PROJECT_ROOT = Path(DEFAULT_DISK_PATH)
TEMPLATE_OPEN_MARKER = "{" + "[<"
TEMPLATE_CLOSE_MARKER = ">" + "]}"


@dataclass(frozen=True)
class RealCodeNode:
    file_url: str
    file_name: str
    source_path: Path | None = None
    content: str | None = None
    enabled: int = 1
    incremented: int = 0

    def read_content(self) -> str:
        if self.content is not None:
            return self.content
        if self.source_path is None:
            raise ValueError(f"source_path is required for {self.file_name}")
        return self.source_path.read_text(encoding="utf-8")


CREATE_BT_STATION_SQL = """-- 班列运输：场站管理
CREATE SEQUENCE cs_bt_station_seq
INCREMENT 1
MINVALUE 1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

CREATE TABLE cs_bt_station (
    id int8 NOT NULL DEFAULT nextval('cs_bt_station_seq'::regclass),
    station_code varchar(64),
    station_name varchar(128),
    station_type varchar(64),
    province_city varchar(128),
    address varchar(255),
    contact_person varchar(64),
    contact_phone varchar(32),
    status varchar(8),
    tenancy varchar(64),
    company_id int8,
    creator int8,
    create_time timestamp,
    modifier int8,
    modify_time timestamp,
    deleted int4 DEFAULT 0,
    CONSTRAINT cs_bt_station_pkey PRIMARY KEY (id)
);

COMMENT ON TABLE cs_bt_station IS '场站管理表';
COMMENT ON COLUMN cs_bt_station.id IS '主键ID';
COMMENT ON COLUMN cs_bt_station.station_code IS '场站编号';
COMMENT ON COLUMN cs_bt_station.station_name IS '场站名称';
COMMENT ON COLUMN cs_bt_station.station_type IS '场站类型';
COMMENT ON COLUMN cs_bt_station.province_city IS '所在省市';
COMMENT ON COLUMN cs_bt_station.address IS '详细地址';
COMMENT ON COLUMN cs_bt_station.contact_person IS '联系人';
COMMENT ON COLUMN cs_bt_station.contact_phone IS '联系电话';
COMMENT ON COLUMN cs_bt_station.status IS '状态 0-停用 1-启用';
COMMENT ON COLUMN cs_bt_station.tenancy IS '租户ID';
COMMENT ON COLUMN cs_bt_station.company_id IS '公司ID';
COMMENT ON COLUMN cs_bt_station.creator IS '创建人ID';
COMMENT ON COLUMN cs_bt_station.create_time IS '创建时间';
COMMENT ON COLUMN cs_bt_station.modifier IS '修改人ID';
COMMENT ON COLUMN cs_bt_station.modify_time IS '修改时间';
COMMENT ON COLUMN cs_bt_station.deleted IS '删除标识';

INSERT INTO common_auth.sys_menu (
    parent_id, menu_name, menu_name_en, menu_url, menu_icon, sort, status, visible,
    menu_source, module_id, type, component, query, keep_alive, subassembly_name,
    remark, tenancy, company_id, creator, create_time, modifier, modify_time,
    menu_code, system_code, deleted
)
VALUES (
    2063000000000000004,
    '场站管理',
    'Station Management',
    'btStation/btStationList',
    NULL,
    5,
    '0',
    0,
    'mtp',
    7,
    1,
    'basic/btStation/btStationList',
    NULL,
    1,
    'btStationList',
    '场站管理',
    -1,
    -1,
    -1,
    NOW(),
    -1,
    NOW(),
    'mtp.menu.bt-transport.btStation.btStationList',
    'mtp',
    0
);

INSERT INTO common_auth.sys_company_menu (
    com_id, company_id, create_time, creator, menu_code, modifier, modify_time, tenancy, visible
)
VALUES (
    2013121927071186945,
    -1,
    TIMESTAMP '2025-10-01 00:00:00',
    -1,
    'mtp.menu.bt-transport.btStation.btStationList',
    -1,
    TIMESTAMP '2025-10-01 00:00:00',
    -1,
    0
);

INSERT INTO leaf_rule (
    name, prefix, date_format, current_value, length, is_restart, reset_cycle,
    remark, tenancy, company_id, creator, creator_name, create_time, modifier,
    modify_time, deleted, type
)
VALUES (
    'BT_STATION_NO',
    'BTST',
    'yyyyMMdd',
    0,
    4,
    'y',
    'day',
    '场站编号',
    -1,
    -1,
    -1,
    NULL,
    TIMESTAMP '2025-10-01 00:00:00',
    -1,
    TIMESTAMP '2025-10-01 00:00:00',
    0,
    'system'
);
"""


REAL_CODE_NODES = [
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-api/src/main/java/com/chinaservices/dsly/basic/module/btStation/domain/",
        "BtStationItem.java",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-api/src/main/java/com/chinaservices/dsly/basic/module/btStation/domain/BtStationItem.java",
    ),
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-api/src/main/java/com/chinaservices/dsly/basic/module/btStation/domain/",
        "BtStationQuery.java",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-api/src/main/java/com/chinaservices/dsly/basic/module/btStation/domain/BtStationQuery.java",
    ),
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-api/src/main/java/com/chinaservices/dsly/basic/module/btStation/domain/",
        "BtStationPageCondition.java",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-api/src/main/java/com/chinaservices/dsly/basic/module/btStation/domain/BtStationPageCondition.java",
    ),
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/module/btStation/model/",
        "BtStation.java",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/module/btStation/model/BtStation.java",
    ),
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/common/constant/",
        "BasicSqlId.java",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/common/constant/BasicSqlId.java",
    ),
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/module/btStation/dao/",
        "BtStationDao.java",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/module/btStation/dao/BtStationDao.java",
    ),
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/module/btStation/service/",
        "BtStationService.java",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/module/btStation/service/BtStationService.java",
    ),
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/module/btStation/controller/",
        "BtStationController.java",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/java/com/chinaservices/dsly/basic/module/btStation/controller/BtStationController.java",
    ),
    RealCodeNode(
        "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/resources/sql-ext/btStation/",
        "btStation_query_getPageList.sql",
        PROJECT_ROOT
        / "c12-mtp-basic-service/c12-mtp-basic-biz/src/main/resources/sql-ext/btStation/btStation_query_getPageList.sql",
    ),
    RealCodeNode("../", "create_btStation.sql", content=CREATE_BT_STATION_SQL),
]


def validate_real_code_nodes() -> None:
    for node in REAL_CODE_NODES:
        if node.source_path is not None and not node.source_path.is_file():
            raise FileNotFoundError(node.source_path)
        content = node.read_content()
        if TEMPLATE_OPEN_MARKER in content or TEMPLATE_CLOSE_MARKER in content:
            raise ValueError(f"template placeholder found in {node.file_name}")
        if "{%" in content or "%}" in content:
            raise ValueError(f"jinja placeholder found in {node.file_name}")


def connect(dsn: str):
    try:
        import psycopg2
    except ImportError as exc:
        raise SystemExit("psycopg2 is required to seed the database") from exc
    return psycopg2.connect(dsn)


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


def upsert_project(cur, args: argparse.Namespace) -> int:
    cur.execute("select id from tb_generate_project where id = %s", (args.project_id,))
    row = cur.fetchone()
    if row:
        cur.execute(
            """
            update tb_generate_project
               set project_name = %s,
                   disk_path = %s,
                   user_name = %s,
                   project_config_id = %s,
                   remark = %s
             where id = %s
            """,
            (
                args.project_name,
                args.disk_path,
                args.user_name,
                args.project_config_id,
                "攀枝花-多式联运-后端 BtStation 真实 CRUD 代码",
                args.project_id,
            ),
        )
        return args.project_id

    cur.execute(
        """
        insert into tb_generate_project(id, project_config_id, project_name, disk_path, remark, user_name)
        values (%s, %s, %s, %s, %s, %s)
        returning id
        """,
        (
            args.project_id,
            args.project_config_id,
            args.project_name,
            args.disk_path,
            "攀枝花-多式联运-后端 BtStation 真实 CRUD 代码",
            args.user_name,
        ),
    )
    return cur.fetchone()[0]


def replace_real_code(cur, project_id: int) -> None:
    cur.execute("select id from tb_generate_project_path where project_id = %s", (project_id,))
    path_ids = [row[0] for row in cur.fetchall()]
    if path_ids:
        cur.execute("delete from tb_generate_project_path_model where path_id = any(%s)", (path_ids,))
    cur.execute("delete from tb_generate_project_path where project_id = %s", (project_id,))
    cur.execute("delete from tb_generate_project_place_holder where project_id = %s", (project_id,))

    for node in REAL_CODE_NODES:
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
            (path_id, node.read_content()),
        )


def seed(args: argparse.Namespace) -> int:
    validate_real_code_nodes()

    if args.dry_run:
        print(f"project_id={args.project_id}")
        print(f"project_name={args.project_name}")
        print(f"disk_path={args.disk_path}")
        print(f"real_code_nodes={len(REAL_CODE_NODES)}")
        for node in REAL_CODE_NODES:
            print(f"- {node.file_url}{node.file_name}")
        return 0

    conn = connect(args.dsn)
    try:
        with conn:
            with conn.cursor() as cur:
                repair_sequences(cur)
                project_id = upsert_project(cur, args)
                replace_real_code(cur, project_id)
                repair_sequences(cur)
        print(f"Seeded real code into project {project_id}: {args.project_name}")
        return 0
    finally:
        conn.close()


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dsn", default=DEFAULT_DSN)
    parser.add_argument("--project-id", type=int, default=DEFAULT_PROJECT_ID)
    parser.add_argument("--project-name", default=DEFAULT_PROJECT_NAME)
    parser.add_argument("--disk-path", default=DEFAULT_DISK_PATH)
    parser.add_argument("--user-name", default="conchi")
    parser.add_argument("--project-config-id", type=int, default=17)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args(argv)
    return seed(args)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
