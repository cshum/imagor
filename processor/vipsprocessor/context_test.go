package vipsprocessor

import (
	"context"
	"testing"
)

func TestContextRotation(t *testing.T) {
	t.Run("rotation flag is local to processing level", func(t *testing.T) {
		// Create parent context
		parentCtx := withContext(context.Background())

		// Set rotation in parent
		setRotate90(parentCtx)
		if !isRotate90(parentCtx) {
			t.Error("expected parent rotation to be set")
		}

		// Create child context (simulates nested image() processing)
		childCtx := withContext(parentCtx)

		// Child should have fresh rotation context (not inherit parent's rotation)
		if isRotate90(childCtx) {
			t.Error("expected child rotation to be false (fresh context)")
		}

		// Parent rotation should still be set
		if !isRotate90(parentCtx) {
			t.Error("expected parent rotation to still be set")
		}

		// Set rotation in child
		setRotate90(childCtx)
		if !isRotate90(childCtx) {
			t.Error("expected child rotation to be set")
		}

		// Parent rotation should be unaffected
		if !isRotate90(parentCtx) {
			t.Error("expected parent rotation to still be set")
		}
	})

	t.Run("resource tracking persists across parent and child", func(t *testing.T) {
		// Create parent context
		parentCtx := withContext(context.Background())

		var parentCalled, childCalled bool

		// Add callback in parent
		contextDefer(parentCtx, func() {
			parentCalled = true
		})

		// Create child context - shares same resource context
		childCtx := withContext(parentCtx)

		// Add callback in child (goes to shared resource context)
		contextDefer(childCtx, func() {
			childCalled = true
		})

		// Done on child calls all callbacks in shared resource context
		contextDone(childCtx)
		if !childCalled {
			t.Error("expected child callback to be called")
		}
		// Parent callback is also called because they share the same resource context
		if !parentCalled {
			t.Error("expected parent callback to be called (shared resource context)")
		}

		// Calling done again on parent is safe (callbacks already cleared)
		contextDone(parentCtx)
	})

	t.Run("rotation toggle works correctly", func(t *testing.T) {
		ctx := withContext(context.Background())

		// Initially false
		if isRotate90(ctx) {
			t.Error("expected rotation to be false initially")
		}

		// Toggle to true
		setRotate90(ctx)
		if !isRotate90(ctx) {
			t.Error("expected rotation to be true after first toggle")
		}

		// Toggle back to false
		setRotate90(ctx)
		if isRotate90(ctx) {
			t.Error("expected rotation to be false after second toggle")
		}
	})

	t.Run("multiple nested levels", func(t *testing.T) {
		// Level 1 (base image)
		level1Ctx := withContext(context.Background())
		setRotate90(level1Ctx)

		// Level 2 (first nested image)
		level2Ctx := withContext(level1Ctx)
		if isRotate90(level2Ctx) {
			t.Error("expected level 2 rotation to be false")
		}
		setRotate90(level2Ctx)

		// Level 3 (second nested image)
		level3Ctx := withContext(level2Ctx)
		if isRotate90(level3Ctx) {
			t.Error("expected level 3 rotation to be false")
		}

		// All levels should maintain their own rotation state
		if !isRotate90(level1Ctx) {
			t.Error("expected level 1 rotation to still be true")
		}
		if !isRotate90(level2Ctx) {
			t.Error("expected level 2 rotation to still be true")
		}
		if isRotate90(level3Ctx) {
			t.Error("expected level 3 rotation to still be false")
		}
	})
}
