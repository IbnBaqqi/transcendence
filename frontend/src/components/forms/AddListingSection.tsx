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
import { useCreateListing, useUploadListingImage } from "../../api/listings";
import { isApiError } from "../../api/client";
import Button from "../objects/Button.tsx";
import { ImageDropzone } from "../objects/ImageDropzone";
import { useImageGallery } from "../../hooks/useImageGallery";

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
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [partial, setPartial] = useState<{ id: string; failed: number } | null>(null);
  const [photoError, setPhotoError] = useState<string | null>(null);
  const {
    images: photos,
    addFiles: addPhotos,
    removeImage: removePhoto,
  } = useImageGallery({
    maxFiles: MAX_LISTING_IMAGES,
    onError: setPhotoError,
  });

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

    // POST /listings/{id}/images needs an id, so the photos can only go once
    // the listing exists. allSettled rather than all: the listing is already
    // created, so one refused file must not look like a failed submission.
    const results = await Promise.allSettled(
      photos.map((photo) => uploadImage.mutateAsync({ listingId: listing.id, file: photo.file })),
    );
    const failed = results.filter((r) => r.status === "rejected").length;

    if (failed > 0) {
      // Nothing to roll back - the listing exists. Saying "failed" here would
      // send someone back to the form to create a duplicate.
      setPartial({ id: listing.id, failed });
      return;
    }

    void navigate(`/listings/${listing.id}`);
  };

  return (
    <Form form={form} onSubmit={handleSubmit} isEditing={true}>
      <div className="space-y-4">
        <div className="space-y-2">
          <div className="space-y-1">
            <h2 className="text-foreground text-lg font-bold">{t("forms.addListing.title")}</h2>
            <FormField name="title" width="max-w-md" validateOnChange />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-lg font-bold">{t("forms.addListing.photos")}</h2>
            <ImageDropzone
              images={photos}
              onFilesSelected={addPhotos}
              onRemove={removePhoto}
              multiple
              emptyMessage={t("dropzone.noPhotos")}
              helperText={t("dropzone.dragDropAddListing")}
            />
            {photoError && <p className="text-berry-500 text-sm">{photoError}</p>}
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-lg font-bold">
              {t("forms.addListing.description")}
            </h2>
            <FormTextArea name="description" />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-lg font-bold">{t("forms.addListing.category")}</h2>
            <CategorySelect
              name="category"
              ariaLabel={t("forms.addListing.category")}
              width="max-w-md"
            />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-lg font-bold">
              {t("forms.addListing.priceQuantity")}
            </h2>
            <div className="flex flex-row gap-4">
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
          disabled={!form.formState.isValid || createListing.isPending || uploadImage.isPending}
        >
          {createListing.isPending || uploadImage.isPending
            ? t("common.saving")
            : t("forms.addListing.save")}
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
