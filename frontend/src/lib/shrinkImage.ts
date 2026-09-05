// A phone camera produces a 12MP, 3-6MB photo, and the upload endpoint caps a
// file at 5 MiB - so the app rejects ordinary photos and leaves the user with
// no way forward, since "resize it yourself" means an image editor on a phone.
// The browser can do it instead: nothing here is required of the user, and the
// server keeps its cap as the backstop it has to be.
//
// The original resolution is not kept. The largest place a listing photo is
// shown is a detail-page gallery, so 1600px is the useful size; full-resolution
// originals would need a server-side pipeline and a second file per image.
const MAX_EDGE = 1600;
const JPEG_QUALITY = 0.82;

export async function shrinkImage(file: File): Promise<File> {
  // Anything the browser cannot decode comes back unchanged, and the caller's
  // existing size check still has the final word.
  if (typeof createImageBitmap !== "function") return file;

  let bitmap: ImageBitmap;
  try {
    // from-image: a photo taken sideways carries its rotation in EXIF, and
    // drawing it to a canvas without this bakes in the wrong orientation.
    bitmap = await createImageBitmap(file, { imageOrientation: "from-image" });
  } catch {
    return file;
  }

  try {
    const scale = Math.min(1, MAX_EDGE / Math.max(bitmap.width, bitmap.height));
    const canvas = document.createElement("canvas");
    canvas.width = Math.round(bitmap.width * scale);
    canvas.height = Math.round(bitmap.height * scale);

    const ctx = canvas.getContext("2d");
    if (!ctx) return file;

    // JPEG has no alpha channel: a transparent PNG composites onto black
    // without a ground to draw on first.
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height);

    const blob = await new Promise<Blob | null>((resolve) =>
      canvas.toBlob(resolve, "image/jpeg", JPEG_QUALITY),
    );
    // A photograph in PNG is several times larger than the same picture as
    // JPEG, so this is usually a large win - but an already-small JPEG can come
    // back bigger, and then the original is the better file.
    if (!blob || blob.size >= file.size) return file;

    return new File([blob], file.name.replace(/\.[^.]+$/, "") + ".jpg", {
      type: "image/jpeg",
      lastModified: file.lastModified,
    });
  } finally {
    bitmap.close();
  }
}
