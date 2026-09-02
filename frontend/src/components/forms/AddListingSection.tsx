import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslation } from "react-i18next";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { CategorySelect } from "./CategorySelect";
import { FormTextArea } from "./FormTextArea";
import { makeAddListingSchema, type AddListingFormValues } from "../../schemas/addListing";
import { useCategories, flattenCategories } from "../../api/categories";
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
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
    // defaultValues: {
    //   firstname: user.firstname ?? "",
    //   lastname: user.lastname ?? "",
    //   phone_number: user.phone_number ?? "",
    //   location: user.location ?? "",
    // },
  });

  const [photoError, setPhotoError] = useState<string | null>(null);
  const {
    images: photos,
    addFiles: addPhotos,
    removeImage: removePhoto,
  } = useImageGallery({
    maxFiles: MAX_LISTING_IMAGES,
    onError: setPhotoError,
  });

  const handleSubmit = (data: AddListingFormValues) => {
    console.log(data, photos);
    // TODO: blocked by #109 Save to API here. A listing has to exist before
    // images can be uploaded (POST /listings/{id}/images needs the new
    // listing's id), so once CreateListing is wired up: create the listing
    // first, then call useUploadListingImage(listing.id) once per file in
    // `photos` (see src/api/listings.ts).
  };
  // TODO: blocked by #109 Add hooks to save data to backend
  // const { handleSubmit: handleSave, isSubmitting, submitError } = useFormSubmit<AddListingFormValues>(
  // async (data) => {
  //   await api.updateAddListing(data); // whatever your API call looks like
  // }

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
        <Button variant="primary" type="submit" disabled={!form.formState.isValid}>
          {/* TODO: blocked by #109 Insert API here */}
          {t("forms.addListing.save")}
        </Button>
      </div>
    </Form>
  );
}
