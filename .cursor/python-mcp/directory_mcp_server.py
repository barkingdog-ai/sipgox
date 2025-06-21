"""目錄服務器 - 提供列印當前資料夾功能"""  # noqa: INP001

from pathlib import Path

from mcp.server.fastmcp import FastMCP

mcp = FastMCP(name="ProjectDirectory")


@mcp.tool(name="print_current_directory")
def print_current_directory() -> str:
    """列印當前 project 根目錄的路徑"""
    return str(Path(__file__).parent.parent.parent)


if __name__ == "__main__":
    mcp.run()
