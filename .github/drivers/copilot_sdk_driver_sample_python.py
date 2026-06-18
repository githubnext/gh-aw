#!/usr/bin/env python3
import asyncio
import json
import os
import sys
from datetime import datetime, timezone

from copilot import CopilotClient, RuntimeConnection
from copilot.session import PermissionHandler
from copilot.session_events import (
    AssistantMessageData,
    SessionEventType,
    ToolExecutionCompleteData,
    ToolExecutionStartData,
)


def read_required_env(name: str) -> str:
    value = os.getenv(name)
    if not value:
        raise RuntimeError(f"{name} is not set")
    return value


def extract_assistant_content(message: object) -> str:
    data = getattr(message, "data", None)
    content = getattr(data, "content", None)
    if isinstance(content, str):
        return content
    direct_content = getattr(message, "content", None)
    if isinstance(direct_content, str):
        return direct_content
    return ""


def _format_timestamp(ts: datetime | None = None) -> str:
    now = ts or datetime.now(timezone.utc)
    return now.isoformat(timespec="milliseconds").replace("+00:00", "Z")


def write_event(event_type: str, data: dict, timestamp: datetime | None = None) -> None:
    entry = {"type": event_type, "timestamp": _format_timestamp(timestamp), "data": data}
    sys.stderr.write(json.dumps(entry) + "\n")


async def main() -> int:
    prompt_path = read_required_env("GH_AW_PROMPT")
    sdk_uri = read_required_env("COPILOT_SDK_URI")
    connection_token = read_required_env("COPILOT_CONNECTION_TOKEN")
    model = read_required_env("COPILOT_MODEL")

    with open(prompt_path, "r", encoding="utf-8") as prompt_file:
        prompt = prompt_file.read()

    client = CopilotClient(
        connection=RuntimeConnection.for_uri(sdk_uri, connection_token=connection_token),
        working_directory=os.getenv("GITHUB_WORKSPACE") or os.getcwd(),
    )

    await client.start()
    session = None
    try:
        session = await client.create_session(on_permission_request=PermissionHandler.approve_all, model=model)

        pending_tool_calls: dict[str, dict[str, str]] = {}

        def handle_event(event) -> None:
            if event.ephemeral:
                return

            match event.type:
                case SessionEventType.USER_MESSAGE:
                    write_event("user.message", {}, event.timestamp)

                case SessionEventType.TOOL_EXECUTION_START:
                    data = event.data
                    if isinstance(data, ToolExecutionStartData):
                        tool_name = data.tool_name or "unknown"
                        mcp_server_name = data.mcp_server_name or ""
                        if data.tool_call_id:
                            pending_tool_calls[data.tool_call_id] = {"toolName": tool_name, "mcpServerName": mcp_server_name}
                        write_event("tool.execution_start", {"toolName": tool_name, "mcpServerName": mcp_server_name}, event.timestamp)

                case SessionEventType.TOOL_EXECUTION_COMPLETE:
                    data = event.data
                    if isinstance(data, ToolExecutionCompleteData):
                        pending = pending_tool_calls.pop(data.tool_call_id, None) if data.tool_call_id else None
                        tool_name = pending.get("toolName") if pending else None
                        if not tool_name:
                            tool_name = data.tool_description.name if data.tool_description else None
                        tool_name = tool_name or "unknown"
                        mcp_server_name = pending.get("mcpServerName", "") if pending else ""
                        write_event("tool.execution_complete", {"toolName": tool_name, "mcpServerName": mcp_server_name, "success": data.success}, event.timestamp)

                case SessionEventType.ASSISTANT_MESSAGE:
                    data = event.data
                    if isinstance(data, AssistantMessageData):
                        write_event("assistant.message", {"content": data.content}, event.timestamp)

        session.on(handle_event)

        response = await session.send_and_wait(prompt)
        content = extract_assistant_content(response)
        if content:
            if content.endswith("\n"):
                sys.stdout.write(content)
            else:
                sys.stdout.write(f"{content}\n")
        return 0
    finally:
        if session is not None:
            await session.disconnect()
        await client.stop()


if __name__ == "__main__":
    try:
        raise SystemExit(asyncio.run(main()))
    except Exception as error:
        sys.stderr.write(f"[copilot-sdk-driver-sample-python] {type(error).__name__}: {error}\n")
        raise SystemExit(1)
