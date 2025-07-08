import { useEffect, useRef } from "react";

export default function TransferRequest({ open, onClose }) {
  const dialogRef = useRef(null);

  // open the dialog
  useEffect(() => {
    const dialog = dialogRef.current;
    if (open && dialog) {
      if (!dialog.open) dialog.showModal();
    } else if (dialog?.open) {
      dialog.close();
    }
  }, [open]);

  return (
    <dialog ref={dialogRef} onClose={onClose}>
      <h2>So and so wants to send you some files</h2>
      <div class="button-row">
        <button class="accept" onClick={onClose}>Accept</button>
        <button class="decline" onClick={onClose}>Decline</button>
      </div>
    </dialog>
  );
}
