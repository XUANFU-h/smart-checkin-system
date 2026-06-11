import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { SpreadsheetFile, Workbook } from "@oai/artifact-tool";

const __filename = fileURLToPath(import.meta.url);
const rootDir = path.resolve(path.dirname(__filename), "..");
const sqlPath = path.join(rootDir, "smart_checkin.sql");
const outputDir = path.join(rootDir, "outputs");
const outputPath = path.join(outputDir, "人员打卡数据导出_复核版.xlsx");

function decodeSqlBuffer(buf) {
  const utf8 = buf.toString("utf8");
  const badChars = (utf8.match(/\uFFFD/g) || []).length;
  return badChars > 10 ? new TextDecoder("gb18030").decode(buf) : utf8;
}

function splitSqlValues(valueText) {
  const values = [];
  let current = "";
  let inString = false;
  for (let i = 0; i < valueText.length; i++) {
    const ch = valueText[i];
    const next = valueText[i + 1];
    if (inString) {
      if (ch === "\\" && i + 1 < valueText.length) {
        current += next;
        i++;
      } else if (ch === "'" && next === "'") {
        current += "'";
        i++;
      } else if (ch === "'") {
        inString = false;
      } else {
        current += ch;
      }
    } else {
      if (ch === "'") {
        inString = true;
      } else if (ch === ",") {
        values.push(convertValue(current.trim()));
        current = "";
      } else {
        current += ch;
      }
    }
  }
  values.push(convertValue(current.trim()));
  return values;
}

function convertValue(raw) {
  if (raw === "" || /^null$/i.test(raw)) return "";
  const numeric = Number(raw);
  return Number.isFinite(numeric) && String(numeric) === raw ? numeric : raw;
}

function extractRows(sql, tableName) {
  const re = new RegExp(String.raw`INSERT INTO \`${tableName}\` VALUES \((.*)\);`, "g");
  const rows = [];
  let match;
  while ((match = re.exec(sql)) !== null) {
    rows.push(splitSqlValues(match[1]));
  }
  return rows;
}

function dateRange(dates) {
  if (!dates.length) return [];
  const sorted = [...new Set(dates)].sort();
  const start = new Date(`${sorted[0]}T00:00:00`);
  const end = new Date(`${sorted[sorted.length - 1]}T00:00:00`);
  const result = [];
  for (let d = start; d <= end; d = new Date(d.getTime() + 86400000)) {
    result.push(d.toISOString().slice(0, 10));
  }
  return result;
}

function safeName(user) {
  return user.realName || user.nickname || "";
}

function setSheetWidth(sheet, widths, rowCount) {
  widths.forEach((width, idx) => {
    sheet.getRangeByIndexes(0, idx, Math.max(rowCount, 1), 1).format.columnWidthPx = width;
  });
}

function formatHeader(range) {
  range.format = {
    fill: "#0F766E",
    font: { bold: true, color: "#FFFFFF" },
    wrapText: true,
    horizontalAlignment: "center",
    verticalAlignment: "center",
    borders: { preset: "all", style: "thin", color: "#D9D9D9" },
  };
}

function formatBody(range) {
  range.format = {
    wrapText: true,
    verticalAlignment: "top",
    borders: { preset: "all", style: "thin", color: "#E5E7EB" },
  };
}

const sql = decodeSqlBuffer(await fs.readFile(sqlPath));
const userRows = extractRows(sql, "user");
const trackRows = extractRows(sql, "track");
const userTrackRows = extractRows(sql, "user_track");
const checkinRows = extractRows(sql, "checkin_record");

const users = userRows.map((r) => ({
  id: Number(r[0]),
  studentId: String(r[1] ?? ""),
  nickname: String(r[2] ?? ""),
  realName: String(r[3] ?? ""),
  className: String(r[4] ?? ""),
  userType: Number(r[8] ?? 0),
  accountStatus: Number(r[9] ?? 0),
  empowermentScore: Number(r[10] ?? 0),
  moralScore: Number(r[11] ?? 0),
}));

const tracks = new Map(trackRows.map((r) => [Number(r[0]), String(r[1] ?? `赛道${r[0]}`)]));
const userTracks = new Map();
for (const r of userTrackRows) {
  const userId = Number(r[1]);
  const trackId = Number(r[2]);
  if (!userTracks.has(userId)) userTracks.set(userId, []);
  userTracks.get(userId).push(trackId);
}

const records = checkinRows.map((r) => ({
  id: Number(r[0]),
  userId: Number(r[1]),
  trackId: Number(r[2]),
  status: Number(r[6] ?? 0),
  score: Number(r[7] ?? 0),
  publishStatus: Number(r[10] ?? 0),
  date: String(r[14] ?? "").slice(0, 10),
  createTime: String(r[15] ?? ""),
}));

const allDates = dateRange(records.map((r) => r.date).filter(Boolean));
const recordMap = new Map();
for (const r of records) {
  if (!r.date) continue;
  const key = `${r.userId}|${r.trackId}|${r.date}`;
  if (!recordMap.has(key)) recordMap.set(key, []);
  recordMap.get(key).push(r);
}

const activeUsers = users
  .filter((u) => u.userType === 0)
  .sort((a, b) => a.className.localeCompare(b.className, "zh-Hans-CN") || safeName(a).localeCompare(safeName(b), "zh-Hans-CN"));

const summaryHeader = ["序号", "姓名", "班级", "长学号", "参与赛道数量", "赋能分", "德育分", "单赛道打卡次数与日期明细", "用户每日打卡数据"];
const trackDetailHeader = ["序号", "姓名", "班级", "长学号", "赛道", "应打卡日期数", "提交打卡次数", "审核通过次数", "已提交日期", "审核通过日期", "未提交日期"];
const dailyHeader = ["序号", "姓名", "班级", "长学号", "日期", "参与赛道数量", "当日提交赛道数", "当日通过赛道数", "已提交赛道", "未提交赛道"];

const summaryRows = [];
const trackDetailRows = [];
const dailyRows = [];

activeUsers.forEach((user, userIndex) => {
  const trackIds = [...new Set(userTracks.get(user.id) || [])].sort((a, b) => a - b);
  const trackNames = trackIds.map((id) => tracks.get(id) || `赛道${id}`);

  const trackSummaries = trackIds.map((trackId, idx) => {
    const submittedDates = [];
    const passedDates = [];
    const missedDates = [];
    for (const date of allDates) {
      const dayRecords = recordMap.get(`${user.id}|${trackId}|${date}`) || [];
      if (dayRecords.length > 0) {
        submittedDates.push(date);
        if (dayRecords.some((r) => r.status === 1)) passedDates.push(date);
      }
      else missedDates.push(date);
    }
    trackDetailRows.push([
      trackDetailRows.length + 1,
      safeName(user),
      user.className,
      user.studentId,
      trackNames[idx],
      allDates.length,
      submittedDates.length,
      passedDates.length,
      submittedDates.join("、") || "无",
      passedDates.join("、") || "无",
      missedDates.join("、") || "无",
    ]);
    return `${trackNames[idx]}：提交${submittedDates.length}次，通过${passedDates.length}次；已提交[${submittedDates.join("、") || "无"}]；未提交[${missedDates.join("、") || "无"}]`;
  });

  const dailySummaries = allDates.map((date) => {
    const submitted = [];
    const passed = [];
    const missed = [];
    trackIds.forEach((trackId, idx) => {
      const dayRecords = recordMap.get(`${user.id}|${trackId}|${date}`) || [];
      if (dayRecords.length > 0) {
        submitted.push(trackNames[idx]);
        if (dayRecords.some((r) => r.status === 1)) passed.push(trackNames[idx]);
      }
      else missed.push(trackNames[idx]);
    });
    dailyRows.push([
      dailyRows.length + 1,
      safeName(user),
      user.className,
      user.studentId,
      date,
      trackIds.length,
      submitted.length,
      passed.length,
      submitted.join("、") || "无",
      missed.join("、") || "无",
    ]);
    return `${date}：提交${submitted.length}/${trackIds.length}，通过${passed.length}/${trackIds.length}`;
  });

  summaryRows.push([
    userIndex + 1,
    safeName(user),
    user.className,
    user.studentId,
    trackIds.length,
    user.empowermentScore,
    user.moralScore,
    trackSummaries.join("\n") || "无赛道数据",
    dailySummaries.join("\n") || "无打卡日期数据",
  ]);
});

const workbook = Workbook.create();
const summarySheet = workbook.worksheets.add("人员汇总");
const trackSheet = workbook.worksheets.add("单赛道明细");
const dailySheet = workbook.worksheets.add("每日打卡数据");

summarySheet.getRangeByIndexes(0, 0, summaryRows.length + 1, summaryHeader.length).values = [summaryHeader, ...summaryRows];
trackSheet.getRangeByIndexes(0, 0, trackDetailRows.length + 1, trackDetailHeader.length).values = [trackDetailHeader, ...trackDetailRows];
dailySheet.getRangeByIndexes(0, 0, dailyRows.length + 1, dailyHeader.length).values = [dailyHeader, ...dailyRows];

for (const [sheet, header, widths] of [
  [summarySheet, summaryHeader, [60, 100, 120, 130, 110, 90, 90, 520, 360]],
  [trackSheet, trackDetailHeader, [60, 100, 120, 130, 120, 110, 110, 110, 420, 420, 520]],
  [dailySheet, dailyHeader, [60, 100, 120, 130, 110, 110, 130, 130, 280, 360]],
]) {
  sheet.freezePanes.freezeRows(1);
  const rowCount = sheet.getUsedRange().rowCount;
  setSheetWidth(sheet, widths, rowCount);
  formatHeader(sheet.getRangeByIndexes(0, 0, 1, header.length));
  const used = sheet.getUsedRange();
  formatBody(used);
  sheet.getRangeByIndexes(0, 0, 1, header.length).format.rowHeightPx = 32;
}

summarySheet.tables.add(summarySheet.getUsedRange(), true, "SummaryTable");
trackSheet.tables.add(trackSheet.getUsedRange(), true, "TrackDetailTable");
dailySheet.tables.add(dailySheet.getUsedRange(), true, "DailyCheckinTable");

await fs.mkdir(outputDir, { recursive: true });
const inspect = await workbook.inspect({
  kind: "table",
  range: "人员汇总!A1:I6",
  include: "values",
  tableMaxRows: 6,
  tableMaxCols: 9,
  maxChars: 3000,
});
console.log(inspect.ndjson);

const rendered = await workbook.render({ sheetName: "人员汇总", range: "A1:I8", scale: 1, format: "png" });
await fs.writeFile(path.join(outputDir, "人员汇总预览.png"), new Uint8Array(await rendered.arrayBuffer()));

const xlsx = await SpreadsheetFile.exportXlsx(workbook);
await xlsx.save(outputPath);

console.log(JSON.stringify({
  outputPath,
  users: activeUsers.length,
  tracks: tracks.size,
  checkinRecords: records.length,
  dateStart: allDates[0] || "",
  dateEnd: allDates.at(-1) || "",
}, null, 2));
