import { useState, useContext } from "react";
import { ReactComponent as CloseIcon } from "./assets/close.svg";
import { ErrorContext } from "./State";

export default function ErrorTray() {
  const { errors, removeError } = useContext(ErrorContext);
  const [removingIndexes, setRemovingIndexes] = useState(new Set());
  const maxVisible = 3;

  const remove = (id) => {
    setRemovingIndexes((prev) => new Set(prev).add(id));
    setTimeout(() => {
      removeError(id);
      setRemovingIndexes((prev) => {
        const copy = new Set(prev);
        copy.delete(id);
        return copy;
      });
    }, 100); // Match CSS transition duration
  };

  return (
    <div className="error-tray">
      {errors.slice(0, maxVisible).reverse().map((error, index) => (
        <div
          key={error.id}
          className={`error-message ${removingIndexes.has(error.id) ? "removing" : ""}`}
          style={{ zIndex: maxVisible - index }}>
          <p>{error.message}</p>
          <button className="error-button" onClick={() => remove(error.id)}>
            <CloseIcon className="error-icon" />
          </button>
        </div>
      ))}
    </div>
  );
}