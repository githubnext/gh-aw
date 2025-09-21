async function uploadAssetsMain() {
    const fs = require("fs");
    const path = require("path");
    const crypto = require("crypto");
    const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";
    const branchName = process.env.GITHUB_AW_ASSETS_BRANCH;
    if (!branchName || typeof branchName !== "string") {
        core.setFailed("GITHUB_AW_ASSETS_BRANCH environment variable is required but not set");
        return;
    }
    core.info(`Using assets branch: ${branchName}`);
    const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
    if (!outputContent) {
        core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
        core.setOutput("upload_count", "0");
        core.setOutput("branch_name", branchName);
        return;
    }
    if (outputContent.trim() === "") {
        core.info("Agent output content is empty");
        core.setOutput("upload_count", "0");
        core.setOutput("branch_name", branchName);
        return;
    }
    core.info(`Agent output content length: ${outputContent.length}`);
    let validatedOutput;
    try {
        validatedOutput = JSON.parse(outputContent);
    }
    catch (error) {
        core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
        return;
    }
    if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
        core.info("No valid items found in agent output");
        core.setOutput("upload_count", "0");
        core.setOutput("branch_name", branchName);
        return;
    }
    const uploadAssetItems = validatedOutput.items.filter(item => item.type === "upload-asset");
    if (uploadAssetItems.length === 0) {
        core.info("No upload-asset items found in agent output");
        core.setOutput("upload_count", "0");
        core.setOutput("branch_name", branchName);
        return;
    }
    core.info(`Found ${uploadAssetItems.length} upload-asset item(s)`);
    let uploadCount = 0;
    let hasChanges = false;
    try {
        try {
            await exec.exec(`git rev-parse --verify origin/${branchName}`);
            await exec.exec(`git checkout -B ${branchName} origin/${branchName}`);
            core.info(`Checked out existing branch from origin: ${branchName}`);
        }
        catch (originError) {
            core.info(`Creating new orphaned branch: ${branchName}`);
            await exec.exec(`git checkout --orphan ${branchName}`);
            await exec.exec(`git rm -rf .`);
            await exec.exec(`git clean -fdx`);
        }
        for (const asset of uploadAssetItems) {
            try {
                const { fileName, sha, size, targetFileName } = asset;
                if (!fileName || !sha || !targetFileName) {
                    core.error(`Invalid asset entry missing required fields: ${JSON.stringify(asset)}`);
                    continue;
                }
                const assetSourcePath = path.join("/tmp/safe-outputs/assets", fileName);
                if (!fs.existsSync(assetSourcePath)) {
                    core.warning(`Asset file not found: ${assetSourcePath}`);
                    continue;
                }
                const fileContent = fs.readFileSync(assetSourcePath);
                const computedSha = crypto.createHash("sha256").update(fileContent).digest("hex");
                if (computedSha !== sha) {
                    core.warning(`SHA mismatch for ${fileName}: expected ${sha}, got ${computedSha}`);
                    continue;
                }
                if (fs.existsSync(targetFileName)) {
                    core.info(`Asset ${targetFileName} already exists, skipping`);
                    continue;
                }
                fs.copyFileSync(assetSourcePath, targetFileName);
                await exec.exec(`git add "${targetFileName}"`);
                uploadCount++;
                hasChanges = true;
                core.info(`Added asset: ${targetFileName} (${size} bytes)`);
            }
            catch (error) {
                core.warning(`Failed to process asset ${asset.fileName}: ${error instanceof Error ? error.message : String(error)}`);
            }
        }
        if (hasChanges) {
            const commitMessage = `[skip-ci] Add ${uploadCount} asset(s)`;
            await exec.exec(`git`, [`commit`, `-m`, `"${commitMessage}"`]);
            if (isStaged) {
                core.summary.addRaw("## Staged Asset Publication");
            }
            else {
                await exec.exec(`git push origin ${branchName}`);
                core.summary
                    .addRaw("## Assets")
                    .addRaw(`Successfully uploaded **${uploadCount}** assets to branch \`${branchName}\``)
                    .addRaw("");
                core.info(`Successfully uploaded ${uploadCount} assets to branch ${branchName}`);
            }
            for (const asset of uploadAssetItems) {
                if (asset.fileName && asset.sha && asset.size && asset.url) {
                    core.summary.addRaw(`- [\`${asset.fileName}\`](${asset.url}) → \`${asset.targetFileName}\` (${asset.size} bytes)`);
                }
            }
            core.summary.write();
        }
        else {
            core.info("No new assets to upload");
        }
    }
    catch (error) {
        core.setFailed(`Failed to upload assets: ${error instanceof Error ? error.message : String(error)}`);
        return;
    }
    core.setOutput("upload_count", uploadCount.toString());
    core.setOutput("branch_name", branchName);
}
(async () => {
    await uploadAssetsMain();
})();

