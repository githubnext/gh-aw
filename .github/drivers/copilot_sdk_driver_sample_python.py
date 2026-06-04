#!/usr/bin/env python3
import asyncio
import os
import sys

from copilot import CopilotClient, RuntimeConnection
from copilot.session import PermissionHandler


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
