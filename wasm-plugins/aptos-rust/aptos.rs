extern crate alloc;
extern crate serde;
extern crate serde_json;
extern crate wee_alloc;
extern crate lazy_static;

use alloc::vec::Vec;
use alloc::string::String;
use core::slice;
use serde::{Deserialize, Serialize};
use log::{info};
use core::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};

#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

// Safe wrapper for raw pointer
struct SafePointer(*mut u8);

unsafe impl Send for SafePointer {}
unsafe impl Sync for SafePointer {}

// Global tracker for allocated memory
lazy_static::lazy_static! {
    static ref ALLOCATED_BLOCKS: Arc<Mutex<Vec<(SafePointer, usize)>>> = Arc::new(Mutex::new(Vec::new()));
    static ref IS_DEALLOCATED: AtomicBool = AtomicBool::new(false);
}

#[repr(C)]
#[derive(Serialize, Deserialize, Debug)]
struct AptosTransaction {
    changes: Vec<Change>,
    events: Vec<Event>,
}

#[repr(C)]
#[derive(Serialize, Deserialize, Debug)]
struct Change {
    #[serde(rename = "type")]
    change_type: String,
    address: String,
}

#[repr(C)]
#[derive(Serialize, Deserialize, Debug)]
struct Event {
    #[serde(rename = "type")]
    event_type: String,
    guid: GUID,
}

#[repr(C)]
#[derive(Serialize, Deserialize, Debug)]
struct GUID {
    account_address: String,
}

#[repr(C)]
#[derive(Serialize, Deserialize, Debug)]
struct Segment {
    suffix: String,
    data: String,
}

#[no_mangle]
pub extern "C" fn process(ptr: *mut u8, size: usize) -> usize {
    info!("process called with ptr: {:?}, size: {}", ptr, size);

    let input_data = unsafe { slice::from_raw_parts(ptr, size) };
    match process_aptos_transaction(input_data) {
        Ok(output_data) => {
            let output_size = output_data.len();
            unsafe {
                core::ptr::copy_nonoverlapping(output_data.as_ptr(), ptr, output_size);
            }
            output_size
        },
        Err(e) => {
            info!("Error processing transaction: {:?}", e);
            0
        }
    }
}

fn process_aptos_transaction(data: &[u8]) -> Result<Vec<u8>, serde_json::Error> {
    let incoming: AptosTransaction = serde_json::from_slice(data)?;
    let mut result = Vec::with_capacity(incoming.changes.len() + incoming.events.len());

    for change in incoming.changes.iter() {
        let event_type = extract_event_type(&change.change_type);
        let address = if change.address.is_empty() { "empty".to_string() } else { change.address.clone() };
        let suffix = format!("event.{}.{}", address, event_type);
        let change_data = serde_json::to_string(change)?;
        result.push(Segment { suffix, data: change_data });
    }

    for event in incoming.events.iter() {
        let event_type = extract_event_type(&event.event_type);
        let address = if event.guid.account_address.is_empty() { "empty".to_string() } else { event.guid.account_address.clone() };
        let suffix = format!("event.{}.{}", address, event_type);
        let event_data = serde_json::to_string(event)?;
        result.push(Segment { suffix, data: event_data });
    }

    if result.is_empty() {
        return Ok(Vec::new());
    }

    serde_json::to_vec(&result)
}

fn extract_event_type(event_type: &str) -> String {
    let parts: Vec<&str> = event_type.split("::").collect();
    let mut event_type = parts.last().unwrap().to_string();
    if let Some(index) = event_type.find('<') {
        event_type = event_type[..index].to_string();
    }
    if let Some(index) = event_type.find('.') {
        event_type = event_type[index + 1..].to_string();
    }
    event_type
}

#[no_mangle]
pub extern "C" fn allocate(size: usize) -> *mut u8 {
    info!("Allocating memory of size: {}", size);
    let mut buf = Vec::with_capacity(size);
    let ptr = buf.as_mut_ptr();
    unsafe {
        // Initialize the allocated memory to zero to avoid undefined behavior
        core::ptr::write_bytes(ptr, 0, size);
    }
    core::mem::forget(buf);

    // Track allocated memory
    let mut allocated_blocks = ALLOCATED_BLOCKS.lock().unwrap();
    allocated_blocks.push((SafePointer(ptr), size));

    info!("Allocated memory at ptr: {:?}", ptr);
    ptr
}

#[no_mangle]
pub extern "C" fn deallocate(ptr: *mut u8, size: usize) {
    info!("Deallocating memory of size: {} at ptr: {:?}", size, ptr);
    // Ensure that the Vec takes ownership of the memory, and it will be freed when it goes out of scope.
    unsafe {
        Vec::from_raw_parts(ptr, size, size);
    }

    // Remove from the tracker
    let mut allocated_blocks = ALLOCATED_BLOCKS.lock().unwrap();
    if let Some(pos) = allocated_blocks.iter().position(|(p, s)| p.0 == ptr && *s == size) {
        allocated_blocks.remove(pos);
    }

    info!("Deallocated memory at ptr: {:?}", ptr);
}

// Function to deallocate all allocated memory blocks
pub fn deallocate_all() {
    if IS_DEALLOCATED.load(Ordering::SeqCst) {
        return;
    }
    let mut allocated_blocks = ALLOCATED_BLOCKS.lock().unwrap();
    for &(SafePointer(ptr), size) in allocated_blocks.iter() {
        unsafe {
            Vec::from_raw_parts(ptr, size, size);
        }
    }
    allocated_blocks.clear();
    IS_DEALLOCATED.store(true, Ordering::SeqCst);
    info!("Deallocated all memory blocks");
}

#[no_mangle]
pub extern "C" fn deallocate_all_memory() {
    deallocate_all();
}
