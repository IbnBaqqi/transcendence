import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router-dom";
import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslation } from "react-i18next";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { CategorySelect } from "./CategorySelect";
import { FormTextArea } from "./FormTextArea";
import { makeAddListingSchema, type AddListingFormValues } from "../../schemas/addListing";
import { useCategories, flattenCategories } from "../../api/categories";
import { useCreateListing, useDeleteListingImage, useUploadListingImage } from "../../api/listings";
import { isApiError } from "../../api/client";
import Button from "../objects/Button.tsx";
import { ImageDropzone } from "../objects/ImageDropzone";
import { useImageGallery, type GalleryImage } from "../../hooks/useImageGallery";
import { describeUploadError } from "../../lib/uploadErrors";

// Mirrors the backend's default MAX_IMAGES_PER_LISTING (see
// backend/internal/config/config.go). The server enforces the real cap on
// upload; this just keeps the UI from letting people queue up more than
// it'll accept.
const MAX_LISTING_IMAGES = 5;

export function AddListingSection() {
  const { t } = useTranslation();
  const { data: categories } = useCategories();
  const schema = useMemo(
    () => makeAddListingSchema(flattenCategories(categories ?? []).map((c) => c.slug)),
    [categories],
  );

  const form = useForm<AddListingFormValues>({
    resolver: zodResolver(schema),
    mode: "onBlur",
  });

  const navigate = useNavigate();
  const createListing = useCreateListing();
  const uploadImage = useUploadListingImage();
  const deleteImage = useDeleteListingImage();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [partial, setPartial] = useState<{ id: string; failed: number } | null>(null);
  const [photoError, setPhotoError] = useState<string | null>(null);
  const {
    images: photos,
    addFiles: addPhotos,
    removeImage: removePhoto,
    moveImage: movePhoto,
    updateImage: updatePhoto,
  } = useImageGallery({
    maxFiles: MAX_LISTING_IMAGES,
    onError: setPhotoError,
  });

  const uploadPhoto = async (listingId: string, photo: GalleryImage) => {
    updatePhoto(photo.id, {
      status: "uploading",
      progress: undefined,
      error: undefined,
      retryable: undefined,
    });
    try {
      const image = await uploadImage.mutateAsync({
        listingId,
        file: photo.file,
        onProgress: (progress) => updatePhoto(photo.id, { progress }),
      });
      updatePhoto(photo.id, { status: "uploaded", serverId: image.id });
      return true;
    } catch (err) {
      const { message, retryable } = describeUploadError(err, photo.file.name);
      updatePhoto(photo.id, { status: "error", error: message, retryable });
      return false;
    }
  };

  const handleSubmit = async (data: AddListingFormValues) => {
    setSubmitError(null);
    setPartial(null);

    let listing;
    try {
      listing = await createListing.mutateAsync(data);
    } catch (err) {
      setSubmitError(isApiError(err) ? err.message : t("forms.addListing.createFailed"));
      return;
    }

    // Sequential: the server assigns position by arrival order
    // (backend/sql/queries/listing_images.sql), so parallel uploads scramble the gallery.
    let failed = 0;
    for (const photo of photos) {
      if (!(await uploadPhoto(listing.id, photo))) failed += 1;
    }

    if (failed > 0) {
      // Nothing to roll back - the listing exists. Saying "failed" here would
      // send someone back to the form to create a duplicate.
      setPartial({ id: listing.id, failed });
      return;
    }

    void navigate(`/listings/${listing.id}`);
  };

  const handleRetry = async (id: string) => {
    const photo = photos.find((p) => p.id === id);
    if (!partial || !photo) return;
    if (!(await uploadPhoto(partial.id, photo))) return;

    // Clearing `partial` instead would re-enable submit and let a second click
    // create the duplicate listing that whole branch exists to prevent.
    if (partial.failed <= 1) void navigate(`/listings/${partial.id}`);
    else setPartial({ ...partial, failed: partial.failed - 1 });
  };

  const handleRemovePhoto = async (id: string) => {
    const photo = photos.find((p) => p.id === id);
    if (!photo?.serverId || !partial) {
      removePhoto(id);
      return;
    }

    setPhotoError(null);
    try {
      await deleteImage.mutateAsync({ listingId: partial.id, imageId: photo.serverId });
      removePhoto(id);
    } catch (err) {
      // Dropping it locally anyway would hide a photo that is still on the listing.
      setPhotoError(isApiError(err) ? err.message : t("common.somethingWentWrong"));
    }
  };

  return (
    <Form form={form} onSubmit={handleSubmit} isEditing={true}>
      <div className="space-y-4">
        <div className="space-y-2">
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">
              {t("forms.addListing.title")}
            </h2>
            <FormField name="title" width="max-w-md" validateOnChange />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">
              {t("forms.addListing.photos")}
            </h2>
            <ImageDropzone
              images={photos}
              onFilesSelected={addPhotos}
              onRemove={handleRemovePhoto}
              onRetry={handleRetry}
              onMove={partial ? undefined : movePhoto}
              multiple
              emptyMessage={t("dropzone.noPhotos")}
              helperText={t("dropzone.dragDropAddListing")}
            />
            {photoError && <p className="text-berry-500 text-sm">{photoError}</p>}
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">
              {t("forms.addListing.description")}
            </h2>
            <FormTextArea name="description" />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">
              {t("forms.addListing.category")}
            </h2>
            <CategorySelect
              name="category"
              ariaLabel={t("forms.addListing.category")}
              width="max-w-md"
            />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">
              {t("forms.addListing.priceQuantity")}
            </h2>
            {/* grid, not flex: a flex item with width:auto sizes to its
                content, so w-full inside it resolves to that same width and
                the pair never shares the row. Grid columns stretch. */}
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField
                name="price"
                label={t("forms.price")}
                placeholder={t("forms.pricePlaceholder")}
                type="number"
                width="max-w-sm"
                validateOnChange
              />
              <FormField
                name="quantity"
                label={t("forms.quantity")}
                type="number"
                width="max-w-sm"
                validateOnChange
              />
            </div>
            <FormField
              name="unit"
              label={t("forms.unit")}
              placeholder={t("forms.unitPlaceholder")}
              width="max-w-md"
              validateOnChange
            />
          </div>
        </div>
        <Button
          variant="primary"
          type="submit"
          // partial means the listing was created and only its photos failed.
          // Leaving submit live would let a second click create the duplicate
          // this whole branch exists to avoid.
          disabled={!form.formState.isValid || form.formState.isSubmitting || partial !== null}
        >
          {form.formState.isSubmitting ? t("common.saving") : t("forms.addListing.save")}
        </Button>

        {submitError && (
          <p role="alert" className="text-berry-500 text-sm">
            {submitError}
          </p>
        )}

        {partial && (
          <p role="alert" className="text-berry-500 space-x-2 text-sm">
            <span>{t("forms.addListing.photosFailed", { count: partial.failed })}</span>
            <Link to={`/listings/${partial.id}`} className="text-accent underline">
              {t("forms.addListing.viewListing")}
            </Link>
          </p>
        )}
      </div>
    </Form>
  );
}
