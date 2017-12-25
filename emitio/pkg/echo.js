process.stdin.setEncoding("utf8");
process.stdin.on("readable", () => {
  const chunk = process.stdin.read();
  if (chunk !== null) {
    // TODO handle buffering
    process.stdout.write(JSON.stringify([chunk]));
  }
});
process.stdin.on("end", () => {
  process.stdout.write("[]");
});
