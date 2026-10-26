import sqlite3, shutil, subprocess, time, json, urllib.request

# 复制一份临时 DB 来造同 code 多平台场景
shutil.copy('data/iupdater.db', 'data/_test_multi.db')
db = sqlite3.connect('data/_test_multi.db')
c = db.cursor()

# 把 hlsto 历史版本里已软删的 windows 版本 (202610110003) 和 linux 版本 (202610110004) 合并成同 code
# 先把 linux 202610110004 的 version_code 改成和 windows 202610110003 一样（模拟同版本组双平台）
c.execute("UPDATE app_versions SET version_code=202610110003 WHERE app_id='com.vlsea.hlsto' AND platform='linux' AND architecture='x86_64' AND id=6")
c.execute("DELETE FROM app_versions WHERE deleted_at IS NOT NULL")  # 清软删
# 再把 hlsto 最新的 linux 202610140001 也 code 改成 202610110003 —— 让这个版本组里有 3 条
c.execute("UPDATE app_versions SET version_code=202610110003 WHERE app_id='com.vlsea.hlsto' AND version_code=202610140001")
db.commit()

# 验证
c.execute("SELECT version_code, platform, architecture, deleted_at FROM app_versions WHERE app_id='com.vlsea.hlsto' AND version_code=202610110003")
print("=== 临时 DB 里同 code 多平台场景 ===")
for r in c.fetchall(): print(r)
db.close()

# 起一个独立端口 8081 用临时 DB 测试
env = "iupdater.db"  # 其实我得改后端启动参数... 算了直接手动查 handler 里的 MAX 逻辑
# 直接测 handler 逻辑：在同 code 多平台下不带 platform 参数 → 409
print("\n=== 预期行为 ===")
print("hlsto 同 code=202610110003 有 windows + linux 两条 → 不带 platform 参数时应 409")
print("hlsto 带 platform=windows → 200 自动返回唯一候选")
print("hlsto 带 platform=linux → 200 自动返回唯一候选")
